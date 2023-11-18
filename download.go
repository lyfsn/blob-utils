package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/urfave/cli"
)

type BlobResponse struct {
	Data []struct {
		Blob string `json:"blob"`
	} `json:"data"`
}

// TODO: Endpoint http para enviar blobs
// TODO: Check full magic header plus randomness
// TODO: Search in next slots until the end
func GetMultiPartBlob(blobChannel chan<- []byte, addr, slot string) error {
	fmt.Println("Retrieving multi-part blob from slot", slot)
	apiURL := addr + "/eth/v1/beacon/blob_sidecars/" + slot

	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Received non-OK status code:", resp.StatusCode)
		return err
	}

	var responseObject BlobResponse
	err = json.NewDecoder(resp.Body).Decode(&responseObject)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}

	for idx, item := range responseObject.Data {
		fmt.Println("Retrieving blob index", idx)
		blobValue := item.Blob

		if blobValue[0:30] != "0x426c6f6273417265436f6d696e67" {
			fmt.Println("Blob number", idx, "does not contain magic header:", blobValue[0:30])
			continue
		}

		hexBytes, err := hex.DecodeString(blobValue[2:])
		if err != nil {
			fmt.Println("Error decoding hex string:", err)
			return err
		}

		blobIndex := hexBytes[17]
		totalBlobs := hexBytes[19]

		//fmt.Printf("Magic header: %v\n", hexBytes[0:32])
		//fmt.Printf("FULL BLOB:\n%v\n", hexBytes)

		// if blobIndex == 0 {
		// 	magicHeaderCustom = hexBytes[]
		// }
		cleanHexBytes := DecodeMagicBlob(hexBytes)

		fmt.Printf("Received blob %d of %d with length=%d\n", blobIndex+1, totalBlobs, len(cleanHexBytes))

		blobChannel <- cleanHexBytes

		filename := slot + "-" + strconv.Itoa(idx) + ".bin"
		err = os.WriteFile(filename, cleanHexBytes, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return err
		}

		fmt.Printf("Hex bytes written to '%s' file successfully.\n", filename)
	}

	//fmt.Println("Total blobs in this slot:", len(responseObject.Data))

	close(blobChannel)
	return nil
}

func DownloadApp(cliCtx *cli.Context) error {

	addr := cliCtx.String(DownloadBeaconRPCURLFlag.Name)
	slot := cliCtx.String(DownloadSlotFlag.Name)

	blobChannel := make(chan []byte)

	go GetMultiPartBlob(blobChannel, addr, slot)

	for {
		select {
		case _, ok := <-blobChannel:
			if !ok {
				fmt.Println("All blobs received.")
				return nil
			}
		}
	}

}
