# indexer

A simple tool for downloading the mentioned paper in your writing space.

## Usage

Just run 

```
indexer -r <your writingspace>
```

The program auto detect the pattern `[indexer/pdf](/attachments/xxxx.xxxxx`.pdf) or `[indexer/pdf](/attachments/xxxx.xxxxx`) in your markdown file.

Then it will download the pdf file to `attachments` directory in your writingspace.

The download pattern in markdown file will be modified to `[indexer/pdf](/attachments/xxxx.xxxxx.pdf)`

If you want keep the program as a daemon (Linux and Macos only), just run:

```
indexer -r <your writingspace> -d
```

It will company with your for writing time sliently. (seems not very smart now)

## proxy 

For magicland, you could use env variable to proxy this downloader. For example: `export https_proxy=http://127.0.0.1:7890;export http_proxy=http://127.0.0.1:7890;export all_proxy=socks5://127.0.0.1:7890` and run `indexer -r <your writingspace>`
