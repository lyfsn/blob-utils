package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TODO: 8 blobs per tx
func encodeBlobs(data []byte, magicHeader []byte) []kzg4844.Blob {
	blobs := []kzg4844.Blob{{}}
	blobIndex := 0
	fieldIndex := 0
	fmt.Printf("---- BEGIN OF BLOB %d -----\n", blobIndex)
	copy(blobs[blobIndex][fieldIndex*32+1:], magicHeader)
	fmt.Printf("%d %d %x || %d\n", 0, fieldIndex, magicHeader, magicHeader)
	fieldIndex++
	for i := 0; i < len(data); i += 31 {
		if fieldIndex == params.BlobTxFieldElementsPerBlob {
			blobs = append(blobs, kzg4844.Blob{})
			blobIndex++
			fieldIndex = 0
			fmt.Printf("---- BEGIN OF BLOB %d -----\n", blobIndex)
		}
		max := i + 31
		if max > len(data) {
			max = len(data)
		}
		//fmt.Println(i, fieldIndex, data[i:max])
		fmt.Printf("%d %d %x || %d\n", i, fieldIndex, data[i:max], data[i:max])
		copy(blobs[blobIndex][fieldIndex*32+1:], data[i:max])
		fieldIndex++
	}
	fmt.Println("Total blobs:", len(blobs))
	return blobs
}

func generateMagicHeader() []byte {
	randomBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(randomBytes, uint64(time.Now().UnixNano()))
	magicHeader := append([]byte{66, 108, 111, 98, 115, 65, 114, 101, 67, 111, 109, 105, 110, 103}, randomBytes...)
	magicHeader = append(magicHeader, []byte{45}...)
	fmt.Printf("Magic header: %v\n", magicHeader)
	return magicHeader
}

func EncodeBlobs(data []byte) ([]kzg4844.Blob, []kzg4844.Commitment, []kzg4844.Proof, []common.Hash, error) {
	var (
		commits         []kzg4844.Commitment
		proofs          []kzg4844.Proof
		versionedHashes []common.Hash
	)

	magicHeader := generateMagicHeader()
	blobs := encodeBlobs(data, magicHeader)

	for _, blob := range blobs {
		commit, err := kzg4844.BlobToCommitment(blob)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		commits = append(commits, commit)

		proof, err := kzg4844.ComputeBlobProof(blob, commit)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		proofs = append(proofs, proof)

		versionedHashes = append(versionedHashes, kZGToVersionedHash(commit))
	}
	return blobs, commits, proofs, versionedHashes, nil
}

var blobCommitmentVersionKZG uint8 = 0x01

// kZGToVersionedHash implements kzg_to_versioned_hash from EIP-4844
func kZGToVersionedHash(kzg kzg4844.Commitment) common.Hash {
	h := sha256.Sum256(kzg[:])
	h[0] = blobCommitmentVersionKZG

	return h
}

func DecodeBlob(blob []byte) []byte {
	if len(blob) != params.BlobTxFieldElementsPerBlob*32 {
		panic("invalid blob encoding")
	}
	var data []byte

	// XXX: the following removes trailing 0s in each field element (see EncodeBlobs), which could be unexpected for certain blobs
	j := 0
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		data = append(data, blob[j:j+31]...)
		j += 32
	}

	i := len(data) - 1
	for ; i >= 0; i-- {
		if data[i] != 0x00 {
			break
		}
	}
	data = data[:i+1]
	return data
}

func DecodeUint256String(hexOrDecimal string) (*uint256.Int, error) {
	var base = 10
	if strings.HasPrefix(hexOrDecimal, "0x") {
		base = 16
	}
	b, ok := new(big.Int).SetString(hexOrDecimal, base)
	if !ok {
		return nil, fmt.Errorf("invalid value")
	}
	val256, nok := uint256.FromBig(b)
	if nok {
		return nil, fmt.Errorf("value is too big")
	}
	return val256, nil
}
