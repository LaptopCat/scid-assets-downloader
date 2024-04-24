package main

import (
	"bytes"
	"log"
	"os"

	"github.com/cristalhq/base64"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

// Download best quality and english localizations by default
var cfg = Config{
	IOSImageQuality:     Images3x,
	AndroidImageQuality: XXXHDPI,
	Locale:              EN,

	DownloadAll: false, // If true, it will ignore above settings and download every file it can see

	CDN:   "https://cdn.id.supercell.com/assets/",
	Retry: 50,
}

type Config struct {
	IOSImageQuality     IOSImageQuality
	AndroidImageQuality AndroidImageQuality
	Locale              Locale
	DownloadAll         bool
	CDN                 string
	Retry               int
}

type IOSImageQuality string

const (
	Images1x IOSImageQuality = "images1x" // lowest res
	Images2x IOSImageQuality = "images2x"
	Images3x IOSImageQuality = "images3x" // highest res
)

type AndroidImageQuality string

const (
	MDPI    AndroidImageQuality = "mdpi" // lowest res
	HDPI    AndroidImageQuality = "hdpi"
	XHDPI   AndroidImageQuality = "xhdpi"
	XXHDPI  AndroidImageQuality = "xxhdpi"
	XXXHDPI AndroidImageQuality = "xxxhdpi" // highest res
)

type Locale string

const (
	AR  Locale = "ar"
	CN  Locale = "cn"
	CNT Locale = "cnt"
	DA  Locale = "da"
	EN  Locale = "de"
	ES  Locale = "es"
	FA  Locale = "fa"
	FR  Locale = "fr"
	HE  Locale = "he"
	ID  Locale = "id"
	IT  Locale = "it"
	JP  Locale = "jp"
	KR  Locale = "kr"
	MS  Locale = "ms"
	NL  Locale = "nl"
	NO  Locale = "no"
	PL  Locale = "pl"
	PT  Locale = "pt"
	RU  Locale = "ru"
	SV  Locale = "sv"
	TH  Locale = "th"
	TR  Locale = "tr"
	VI  Locale = "vi"
)

var locales = []Locale{AR, CN, CNT, DA, EN, ES, FA, FR, HE, ID, IT, JP, KR, MS, NL, NO, PL, PT, RU, SV, TH, TR, VI}
var androidimgqualities = []AndroidImageQuality{MDPI, HDPI, XHDPI, XXHDPI, XXXHDPI}
var iosimgqualities = []IOSImageQuality{Images1x, Images2x, Images3x}

var json = jsoniter.ConfigFastest

var ender = make(chan struct{})
var success, fail, total = 0, 0, 0

var fh = fasthttp.Client{
	MaxConnsPerHost: 9999999,
}

type AssetHashmap map[string]map[string]string

type AssetManifest struct {
	RemoteAssets struct {
		Android       AssetHashmap
		IOS           AssetHashmap `json:"iOS"`
		Localizations AssetHashmap
	}
}

var raw []byte
var parsed AssetManifest

var oldRaw []byte
var oldParsed AssetManifest

func DownloadAsset(t, c, n, h string) (err error) {
	for i := 0; i < cfg.Retry; i++ {
		err = downloadAsset(t, c, n)
		if err == nil {
			success++
			count := success + fail
			log.Printf("Downloaded asset: %s (%d/%d)", n, count, total)

			if count == total {
				ender <- struct{}{}
			}
			return
		}

		log.Println("retry", n, total, i, err)
	}

	fail++
	log.Printf("error while downloading %s: %s", n, err.Error())
	return
}

func downloadAsset(t, c, n string) (err error) {
	route1 := t + "/" + c
	route := route1 + "/" + n
	req := PrepareReq(route)
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = fh.Do(req, resp)
	if err != nil {
		return
	}

	data, err := resp.BodyUncompressed()
	if err != nil {
		return
	}

	os.MkdirAll("assets/"+route1, 0777)
	return os.WriteFile("assets/"+route, data, 0777)
}

func PrepareReq(url string) *fasthttp.Request {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(cfg.CDN + url)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	return req
}

func GetAssetManifest() (err error) {
	req := PrepareReq("AssetManifest.jwt")
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = fh.Do(req, resp)
	if err != nil {
		return err
	}

	raw, err = resp.BodyUncompressed()
	if err != nil {
		return
	}

	raw = bytes.Split(raw, []byte("."))[1] // carve out data from jwt
	raw, err = base64.RawStdEncoding.DecodeToBytes(raw)
	if err != nil {
		return
	}

	if bytes.Equal(raw, oldRaw) {
		log.Fatal("Manifest has not changed, exiting!")
	}

	return json.Unmarshal(raw, &parsed)
}

func CheckAssets[T ~string](new, old AssetHashmap, disallowedCategories []T, exception T, n string) {
	for k, m := range new {
		log.Println("a")
		f := false
		for _, imgk := range disallowedCategories {
			if k == string(imgk) {
				f = true
				break
			}
		}
		if f && k != string(exception) {
			log.Println("skipping", k)
			continue
		}

		oldm, ok := old[k]
		if !ok {
			log.Println("found new category", k)
			oldm = make(map[string]string)
		}
		for name, hash := range m {
			oldhash, ok := oldm[name]
			if !ok {
				log.Printf("New asset: %s (%s)", name, hash)
			}

			if oldhash != hash {
				if ok {
					log.Printf("Asset changed: %s (%s -> %s)\n", name, oldhash, hash)
				}

				total++
				go DownloadAsset(n, k, name, hash)
			}
		}
	}
}

func CheckAssetsPure(old, new AssetHashmap, n string) {
	for k, m := range new {
		oldm, ok := old[k]
		if !ok {
			log.Println("found new category", k)
			oldm = make(map[string]string)
		}
		for name, hash := range m {
			oldhash, ok := oldm[name]
			if !ok {
				log.Printf("New asset: %s (%s)", name, hash)
			}

			if oldhash != hash {
				if ok {
					log.Printf("Asset changed: %s (%s -> %s)\n", name, oldhash, hash)
				}

				total++
				go DownloadAsset(n, k, name, hash)
			}
		}
	}
}

func main() {
	log.Println("Loading old manifest...")
	var err error
	oldRaw, err = os.ReadFile("assets/AssetManifest.json")
	if err != nil {
		log.Fatal("cant load old manifest:", err)
	}

	log.Println("Getting new manifest...")
	err = GetAssetManifest()
	if err != nil {
		log.Fatal("cant get asset manifest:", err)
	}

	log.Println("Parsing old manifest...")
	err = json.Unmarshal(oldRaw, &oldParsed)
	if err != nil {
		log.Fatal("cant parse old manifest:", err)
	}

	if cfg.DownloadAll {
		log.Println("Checking android assets...")
		go CheckAssetsPure(parsed.RemoteAssets.Android, oldParsed.RemoteAssets.Android, "Android")

		log.Println("Checking ios assets...")
		go CheckAssetsPure(parsed.RemoteAssets.IOS, oldParsed.RemoteAssets.IOS, "iOS")

		log.Println("Checking localizations...")
		go CheckAssetsPure(parsed.RemoteAssets.Localizations, oldParsed.RemoteAssets.Localizations, "Localizations")
	} else {
		log.Println("Checking android assets...")
		go CheckAssets(parsed.RemoteAssets.Android, oldParsed.RemoteAssets.Android, androidimgqualities, cfg.AndroidImageQuality, "Android")

		log.Println("Checking ios assets...")
		go CheckAssets(parsed.RemoteAssets.IOS, oldParsed.RemoteAssets.IOS, iosimgqualities, cfg.IOSImageQuality, "iOS")

		log.Println("Checking localizations...")
		go CheckAssets(parsed.RemoteAssets.Localizations, oldParsed.RemoteAssets.Localizations, locales, cfg.Locale, "Localizations")
	}

	<-ender
	log.Printf("All assets downloaded! (%d successful, %d failed, %d total)", success, fail, total)
	log.Println("Saving new manifest...")
	err = os.WriteFile("assets/AssetManifest.json", raw, 0777)
	if err != nil {
		log.Fatal("cant save new manifest:", err)
	}
}
