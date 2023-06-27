package main

import (
	"encoding/hex"
	"fmt"

	goecies "github.com/ecies/go"
)

func main() {
	var pubkeyHex string
	fmt.Print("Enter the ecies pubkey: ")
	fmt.Scanf("%s", &pubkeyHex)
	pubkey, err := goecies.NewPublicKeyFromHex(pubkeyHex)
	if err != nil {
		fmt.Println("Cannot decode ecies pubkey")
		panic(err)
	}

	var bchWif string
	fmt.Print("Enter the BCH private key (WIF): ")
	fmt.Scanf("%s", &bchWif)

	var sbchKey string
	fmt.Print("Enter the sBCH private key (HEX): ")
	fmt.Scanf("%s", &sbchKey)

	encryptedBchWif, err := goecies.Encrypt(pubkey, []byte(bchWif))
	if err != nil {
		fmt.Println("Cannot encrypt BCH key")
		panic(err)
	}
	encryptedSbchKey, err := goecies.Encrypt(pubkey, []byte(sbchKey))
	if err != nil {
		fmt.Println("Cannot encrypt sBCH key")
		panic(err)
	}

	fmt.Printf("The encrypted BCH WIF: %s\n, sBCH key: %s\n",
		hex.EncodeToString(encryptedBchWif),
		hex.EncodeToString(encryptedSbchKey))
}
