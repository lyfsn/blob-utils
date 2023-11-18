# Blobtoss

Made during Devconnect '23 @ Istanbul

CLI for messing around with multi-part blobs. Based on blob-utils by Inphi.

To get started run:

```
git clone https://github.com/PabloCastellano/blob-utils && cd blob-utils
go build
./blob-utils -help
```

## Features

- Upload multi-part blobs to beacon chain (`blob-utils tx`)
- Retrieve multi-part blobs from beacon chain (`blob-utils download`)
- Serve multi-part blobs from beacon chain (`blob-utils serve`)

```
blob-utils tx --private-key PRIV_KEY --chain-id 7011893061 --priority-gas-price 6000000000 --gas-price 800000000000 --max-fee-per-blob-gas 70000000000 --rpc-url http://10.128.0.8:8545 --to 0x0000000000000000000000000000000000000000 --blob-file DoD.jpg
./blob-utils download --slot 129252
```

## Credits

* https://github.com/Inphi/blob-utils