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

type ResponseStruct struct {
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

func DownloadApp(cliCtx *cli.Context) error {

	slot := cliCtx.String(DownloadSlotFlag.Name)
	apiURL := "http://10.128.0.8:5052/eth/v1/beacon/blob_sidecars/" + slot

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

	var responseObject ResponseStruct
	err = json.NewDecoder(resp.Body).Decode(&responseObject)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}

	for idx, item := range responseObject.Data {
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
	return nil
}
