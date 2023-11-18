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

// REALLY SLOW
func removePairsOf00(hexString string) string {
	var result string

	for i := 0; i < len(hexString); i += 2 {
		// Check if the pair of characters is not "00" and append to the result
		if i+1 < len(hexString) && hexString[i:i+2] != "00" {
			result += hexString[i : i+2]
		}
	}

	return result
}

func GetMultiPartBlob(blobChannel chan<- []byte, doneChannel chan<- struct{}, addr, slot string) error {
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
		//fmt.Printf("Value of 'blob': %s\n", blobValue)

		//blobValue = strings.Replace(blobValue, "00", "", -1)
		blobValue = removePairsOf00(blobValue)

		hexBytes, err := hex.DecodeString(blobValue[68:])
		if err != nil {
			fmt.Println("Error decoding hex string:", err)
			return err
		}

		blobChannel <- hexBytes

		// Print the result
		//fmt.Printf("%v", hexBytes)

		filename := slot + "-" + strconv.Itoa(idx) + ".bin"
		err = os.WriteFile(filename, hexBytes, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return err
		}

		fmt.Println("Hex bytes written to '", filename, "' file successfully.")
	}

	//fmt.Println("Total blobs in this slot:", len(responseObject.Data))

	close(doneChannel)
	return nil
}

func DownloadApp(cliCtx *cli.Context) error {

	addr := cliCtx.String(DownloadBeaconRPCURLFlag.Name)
	slot := cliCtx.String(DownloadSlotFlag.Name)

	blobChannel := make(chan []byte)
	doneChannel := make(chan struct{})

	err := GetMultiPartBlob(blobChannel, doneChannel, addr, slot)
	if err != nil {
		fmt.Println("GetMultiPartBlob failed:", err)
		return err
	}

	for {
		select {
		case result, ok := <-blobChannel:
			if !ok {
				// resultChannel is closed, break out of the loop
				break
			}
			fmt.Println("Result:", result)

		case <-doneChannel:
			// doneChannel is closed, exit the loop
			fmt.Println("All results received.")
			return nil
		}
	}

}
