package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type FullBlobStruct struct {
	Sidecar         types.BlobTxSidecar
	VersionedHashes []common.Hash
}

// TODO: maximum is 8 blobs per tx
// TODO: send multiple txs
// TODO: max 255 blobs so that we can serialize it, That is ~32MB in total.
// TODO: Calculate total cost of sending file (single or multipart)
func encodeMultipartBlobs(data []byte, totalBlobs int64) []kzg4844.Blob {
	blobs := []kzg4844.Blob{{}}
	blobIndex := 0
	fieldIndex := 0
	fmt.Printf("---- BEGIN OF BLOB %d -----\n", blobIndex)

	magicHeader := generateMagicHeader(blobIndex+1, totalBlobs)
	copy(blobs[blobIndex][fieldIndex*32:], magicHeader)
	fmt.Printf("%d %d %x || %d\n", 0, fieldIndex, magicHeader, magicHeader)
	fieldIndex++
	for i := 0; i < len(data); i += 31 {

		if fieldIndex == params.BlobTxFieldElementsPerBlob {
			blobs = append(blobs, kzg4844.Blob{})
			blobIndex++
			fieldIndex = 0
			fmt.Printf("---- BEGIN OF BLOB %d -----\n", blobIndex)

			magicHeader := generateMagicHeader(blobIndex+1, totalBlobs)
			copy(blobs[blobIndex][fieldIndex*32:], magicHeader)
			fmt.Printf("%d %d %x || %d\n", 0, fieldIndex, magicHeader, magicHeader)
			fieldIndex++
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

func encodeBlobs(data []byte) []kzg4844.Blob {
	blobs := []kzg4844.Blob{{}}
	blobIndex := 0
	fieldIndex := -1
	for i := 0; i < len(data); i += 31 {
		fieldIndex++
		if fieldIndex == params.BlobTxFieldElementsPerBlob {
			blobs = append(blobs, kzg4844.Blob{})
			blobIndex++
			fieldIndex = 0
		}
		max := i + 31
		if max > len(data) {
			max = len(data)
		}
		copy(blobs[blobIndex][fieldIndex*32+1:], data[i:max])
	}
	return blobs
}

// magicHeader is a 32 bytes array containing a string we use to identify files splitted in multiple blobs
// plus blobIndex and totalBlobs
func generateMagicHeader(blobIndex int, totalBlobs int64) []byte {

	randomBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(randomBytes, uint64(time.Now().UnixNano()))
	magicHeader := make([]byte, 32)
	fmt.Printf("Len1: %d\n", len(magicHeader))

	copy(magicHeader, []byte{66, 108, 111, 98, 115, 65, 114, 101, 67, 111, 109, 105, 110, 103, 46, 1, 46, byte(blobIndex), 46, byte(totalBlobs), 46, 46, 46, 46})
	copy(magicHeader[24:], randomBytes)
	magicHeader = append(magicHeader, []byte{45}...)
	fmt.Printf("Len4: %d\n", len(magicHeader))

	fmt.Printf("Magic header (len=%d): %v\n", len(magicHeader), magicHeader)
	return magicHeader
}

func EncodeMultipartBlob(blobChannel chan<- FullBlobStruct, doneChannel chan<- struct{}, data []byte) {
	fileSize := len(data)
	totalBlobs := fileSize / 131072
	remainder := fileSize % 131072

	if remainder > 0 {
		totalBlobs += 1
	}
	fmt.Printf("File size is %d bytes and will be split into %d blobs of 128KB\n", fileSize, totalBlobs)

	if totalBlobs > 8 {
		fmt.Printf("More than 8 blobs is supported at the moment")
		close(blobChannel)
		//close(doneChannel)
		return
	}

	// blobChannel := make(chan FullBlobStruct)

	// Split in 128KB - 32 bytes (magic header)
	// 131072 - 32 = 131040 bytes
	blobIndex := 0
	for i := 0; i < len(data); i += 131040 {
		chunk := data[i*blobIndex : i*blobIndex+131040]
		sidecar, versionedHashes, err := EncodeBlobs(chunk)
		if err != nil {
			log.Fatalf("failed to compute commitments: %v", err)
		}
		blobStruct := FullBlobStruct{Sidecar: *sidecar, VersionedHashes: versionedHashes}
		blobChannel <- blobStruct

	}

}

func EncodeBlobs(data []byte) (*types.BlobTxSidecar, []common.Hash, error) {
	var (
		blobs           = encodeBlobs(data)
		commits         []kzg4844.Commitment
		proofs          []kzg4844.Proof
		versionedHashes []common.Hash
	)

	for _, blob := range blobs {
		commit, err := kzg4844.BlobToCommitment(blob)
		if err != nil {
			return nil, nil, err
		}
		commits = append(commits, commit)

		proof, err := kzg4844.ComputeBlobProof(blob, commit)
		if err != nil {
			return nil, nil, err
		}
		proofs = append(proofs, proof)

		versionedHashes = append(versionedHashes, kZGToVersionedHash(commit))
	}
	sidecar := types.BlobTxSidecar{
		Blobs:       blobs,
		Commitments: commits,
		Proofs:      proofs,
	}
	return &sidecar, versionedHashes, nil
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
