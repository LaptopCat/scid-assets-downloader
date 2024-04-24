# Supercell ID Assets Downloader
Asset downloader for the in-game Supercell ID panel that is in Supercell games

- Fast (using concurrency)
- Download images, video, audio, localizations (you can select target quality of images, strings locale or just download everything)
- Only downloads updated files (if old manifest is present)

# Usage
0. Install [Go](https://go.dev)
1. Clone this repository
   
> for example, you can do it like this on command line: `git clone https://github.com/LaptopCat/scid-assets-downloader`
2. Download required modules
> `go mod download`
3. Configure (optional)
> Below the top of the `main.go` file, there is a `cfg` variable, in which you can edit the config
4. Build the program
> `go build main.go`
> This drops a `main` binary in the project folder
5. Run it
> `./main`

# Disclaimer
This material is unofficial and is not endorsed by Supercell. For more information see [Supercell's Fan Content Policy](https://www.supercell.com/fan-content-policy)
