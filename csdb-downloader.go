// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]
// ------------------------------------------------------------------------------------------------
// csdb downloader by DKT/Samar
// ------------------------------------------------------------------------------------------------
// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
)

// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]
// ------------------------------------------------------------------------------------------------
// Zmienne globalne
// ------------------------------------------------------------------------------------------------
// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]

var cacheDir string
var sep string

// RssItem - pojednyczy wpis w XML
// ------------------------------------------------------------------------------------------------
type RssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

// XMLRssFeed - tabela XML
// ------------------------------------------------------------------------------------------------
type XMLRssFeed struct {
	Items []RssItem `xml:"channel>item"`
}

// XMLHandle - kto jest autorem wydał
// ------------------------------------------------------------------------------------------------
type XMLHandle struct {
	ID        string `xml:"ID"`
	XMLHandle string `xml:"Handle"`
}

// XMLGroup - kto jest autorem wydał
// ------------------------------------------------------------------------------------------------
type XMLGroup struct {
	ID   string `xml:"ID"`
	Name string `xml:"Name"`
}

// XMLReleasedBy - kto wydał
// ------------------------------------------------------------------------------------------------
type XMLReleasedBy struct {
	XMLHandle []XMLHandle `xml:"Handle"`
	XMLGroup  []XMLGroup  `xml:"Group"`
}

// XMLCredit - XMLCredit za produkcję
// ------------------------------------------------------------------------------------------------
type XMLCredit struct {
	CreditType string    `xml:"CreditType"`
	XMLHandle  XMLHandle `xml:"Handle"`
}

// XMLDownloadLink - download links
// ------------------------------------------------------------------------------------------------
type XMLDownloadLink struct {
	Link string `xml:"Link"`
}

// XMLEvent - kompo
// ------------------------------------------------------------------------------------------------
type XMLEvent struct {
	ID   string `xml:"ID"`
	Name string `xml:"Name"`
}

// XMLReleasedAt - kompa
// ------------------------------------------------------------------------------------------------
type XMLReleasedAt struct {
	XMLEvent XMLEvent `xml:"Event"`
}

// XMLUsedSID - SIDy
// ------------------------------------------------------------------------------------------------
type XMLUsedSID struct {
	ID       string `xml:"ID"`
	HVSCPath string `xml:"HVSCPath"`
	Name     string `xml:"Name"`
	Author   string `xml:"Author"`
}

// XMLRelease - wydanie produkcji na csdb
// ------------------------------------------------------------------------------------------------
type XMLRelease struct {
	ReleaseID         string            `xml:"Release>ID"`
	ReleaseName       string            `xml:"Release>Name"`
	ReleaseType       string            `xml:"Release>Type"`
	ReleaseYear       string            `xml:"Release>ReleaseYear"`
	ReleaseMonth      string            `xml:"Release>ReleaseMonth"`
	ReleaseDay        string            `xml:"Release>ReleaseDay"`
	ReleaseScreenShot string            `xml:"Release>ScreenShot"`
	Rating            float32           `xml:"Release>Rating"`
	XMLReleasedBy     XMLReleasedBy     `xml:"Release>ReleasedBy"`
	XMLReleasedAt     XMLReleasedAt     `xml:"Release>ReleasedAt"`
	Credits           []XMLCredit       `xml:"Release>Credits>Credit"`
	DownloadLinks     []XMLDownloadLink `xml:"Release>DownloadLinks>DownloadLink"`
	UsedSIDs          []XMLUsedSID      `xml:"Release>UsedSIDs>SID"`
}

// LatestRelease - najwyższy numer ID
// ------------------------------------------------------------------------------------------------
type LatestRelease struct {
	ID int `xml:"LatestReleaseId"`
}

// Release - wydanie produkcji na csdb
// ================================================================================================
type Release struct {
	ReleaseID         int
	ReleaseYear       int
	ReleaseMonth      int
	ReleaseDay        int
	ReleaseName       string
	ReleaseType       string
	ReleaseScreenShot string
	ReleasedAt        string
	SIDPath           string
	Rating            float32
	ReleasedBy        []string
	Credits           []string
	DownloadLinks     []string
}

// Config - info o ostatniej ściągniętej produkcji
// ================================================================================================
type Config struct {
	DownloadDirectory string
	NoCompoDirectory  string
	LastID            int
	Date              string
	Types             []string
}

// releases - glówna i globalna tablica z aktualnymi produkcjami
// ================================================================================================
var config Config

// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]
// ------------------------------------------------------------------------------------------------
// Funkcje
// ------------------------------------------------------------------------------------------------
// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]

// ErrCheck - obsługa błedów
// ================================================================================================
func ErrCheck(errNr error) bool {
	if errNr != nil {
		log.Println(errNr)
		return false
	}
	return true
}

// ErrCheck2 - obsługa błedów bez komunikatu
// ================================================================================================
func ErrCheck2(errNr error) bool {
	return errNr == nil
}

// ReadConfig - Odczyt konfiguracji
// ================================================================================================
func ReadConfig() {
	file, _ := os.ReadFile("csdb-downloader.json")
	_ = json.Unmarshal([]byte(file), &config)
}

// WriteConfig - Zapis konfiguracji
// ================================================================================================
func WriteConfig() {
	file, _ := json.MarshalIndent(config, "", " ")
	_ = os.WriteFile("csdb-downloader.json", file, 0666)
}

// fileExists - sprawdzenie czy plik istnieje
// ================================================================================================
func fileExists(filename string) bool {

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
// ================================================================================================
func DownloadFile(path string, filename string, url string) error {

	var err error

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
		os.Chmod(path, 0777)
	}
	if err != nil {
		return err
	}

	filepath := path + sep + filename

	log.Println("Downloading new file " + url)
	log.Println("Writing to " + filepath)

	httpClient := http.Client{
		Timeout: time.Second * 5, // Timeout after 5 seconds
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	out.Close()

	if strings.Contains(strings.ToLower(filename), ".zip") {
		zipReader, err := zip.OpenReader(filepath)
		if ErrCheck(err) {
			defer zipReader.Close()
			for _, file := range zipReader.File {

				log.Println("Extracting from ZIP: " + file.Name)
				if !file.FileInfo().IsDir() {

					outputFile, err := os.OpenFile(
						path+sep+file.Name,
						os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
						file.Mode(),
					)
					if ErrCheck(err) {
						defer outputFile.Close()

						// log.Println("Opening: " + file.Name)
						// log.Println("Output: " + path + sep + file.Name)

						zippedFile, err := file.Open()
						if ErrCheck(err) {
							defer zippedFile.Close()
							log.Println("Writing extracted file " + path + sep + file.Name)
							_, err = io.Copy(outputFile, zippedFile)
							ErrCheck(err)
						}
					}
				}
			}
		}
	}

	return err
}

// makeCharsetReader - decode reader
// ================================================================================================
func makeCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	return input, nil
}

// CSDBPrepareData - parametry (gobackID, startingID, date) - Wątek odczygtujący wszystkie releasy z csdb
// ================================================================================================
func CSDBPrepareData(gobackID int, startingID int, date string, all bool) {

	// log.Println(*date)
	parsedDate, _ := time.Parse("2006-01-02", date)
	// log.Println(parsedDate)

	// lastDate := time.Now().AddDate(0, -historyMaxMonths, 0)
	// lastDate := time.Date(config.HistoryYear, time.Month(config.HistoryMonth), 1, 0, 0, 0, 0, time.Local)

	// pobranie ostatniego release'u
	netClient := &http.Client{Timeout: time.Second * 5}
	resp, err := netClient.Get("https://csdb.dk/webservice/?type=release&id=0")

	if ErrCheck(err) {

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		ErrCheck(err)
		// log.Println(string(body))
		resp.Body.Close()

		// Przerobienie na strukturę

		var entry LatestRelease
		reader := bytes.NewReader(body)
		decoder := xml.NewDecoder(reader)
		decoder.CharsetReader = makeCharsetReader
		err = decoder.Decode(&entry)
		ErrCheck(err)

		// ustalenie od którego zaczynamy
		newestCSDbID := entry.ID
		if config.LastID == 0 && gobackID == 0 {
			config.LastID = newestCSDbID - 64
			log.Println("Running for a first time, downloading 64 last releases. Change your config.json file to adjust the number or use parameters.")
		}
		if gobackID > 0 {
			config.LastID = newestCSDbID - gobackID
		}
		if startingID > 0 {
			config.LastID = startingID
		}

		lastDownloadedID := config.LastID

		log.Println("Checking...")
		log.Println("Newest ID on CSDb is " + strconv.Itoa(newestCSDbID))
		log.Println("Starting with ID " + strconv.Itoa(lastDownloadedID))

		// zaczynamy od ostatniego zawsze, nawet jeżeli robimy tylko update bo może ktoś update'ował dane
		checkingID := lastDownloadedID

		searching := true
		for searching {

			resp, err := netClient.Get("https://csdb.dk/webservice/?type=release&id=" + strconv.Itoa(checkingID))

			if ErrCheck(err) {
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				defer resp.Body.Close()

				if ErrCheck(err) {
					resp.Body.Close()
					// log.Println(string(body))

					// Przerobienie na strukturę

					var entry XMLRelease
					reader := bytes.NewReader(body)
					decoder := xml.NewDecoder(reader)
					decoder.CharsetReader = makeCharsetReader
					err = decoder.Decode(&entry)
					if ErrCheck2(err) {

						// prodTime := time.Date(prodYear, time.Month(prodMonth), prodDay, 0, 0, 0, 0, time.Local)

						// Szukamy takiego release w naszej bazie
						//
						typeOK := false
						for _, relType := range config.Types {
							if relType == entry.ReleaseType {
								typeOK = true
								break
							}
						}

						// Jeżeli nie ma podanych typów to ściągamy wszystko
						if len(config.Types) == 0 || all {
							typeOK = true
						}

						// Info
						prodYear, _ := strconv.Atoi(entry.ReleaseYear)
						prodMonth, _ := strconv.Atoi(entry.ReleaseMonth)
						prodDay, _ := strconv.Atoi(entry.ReleaseDay)

						if prodYear == 0 {
							prodYear = 1982
						}
						if prodMonth == 0 {
							prodMonth = 1
						}
						if prodDay == 0 {
							prodDay = 1
						}

						noDate := prodYear == 1982 && prodMonth == 1 && prodDay == 1
						relDate := time.Date(prodYear, time.Month(prodMonth), prodDay, 0, 0, 0, 1, time.UTC)
						dateProvided := date == ""

						if typeOK {
							color.LightGreen.Printf("ID %d %04d-%02d-%02d %s\n", checkingID, prodYear, prodMonth, prodDay, entry.ReleaseType)
						} else {
							color.Secondary.Printf("ID %d %04d-%02d-%02d %s\n", checkingID, prodYear, prodMonth, prodDay, entry.ReleaseType)
						}

						// Jeżeli typ OK to działamy dalej
						if typeOK {

							// Tworzymy nowy obiekt release który dodamy do slice
							//
							var newRelease Release
							id, _ := strconv.Atoi(entry.ReleaseID)
							newRelease.ReleaseID = id
							newRelease.ReleaseName = entry.ReleaseName
							newRelease.ReleaseScreenShot = entry.ReleaseScreenShot
							newRelease.Rating = entry.Rating
							newRelease.ReleaseYear = prodYear
							newRelease.ReleaseMonth = prodMonth
							newRelease.ReleaseDay = prodDay
							newRelease.ReleaseType = entry.ReleaseType
							newRelease.ReleasedAt = entry.XMLReleasedAt.XMLEvent.Name

							log.Println("Date parsed = " + parsedDate.String())
							log.Println("Date releas = " + relDate.String())

							if relDate.After(parsedDate) || (noDate && dateProvided) {
								if len(entry.UsedSIDs) == 1 {
									newRelease.SIDPath = entry.UsedSIDs[0].HVSCPath
								}

								log.Println("Entry name: " + entry.ReleaseName)
								// log.Println("ID:     ", entry.ReleaseID)
								log.Println("Type: " + entry.ReleaseType)
								// log.Println("Event:  ", entry.XMLReleasedAt.XMLEvent.Name)

								for _, group := range entry.XMLReleasedBy.XMLGroup {
									log.Println("Released by: " + group.Name)
									newRelease.ReleasedBy = append(newRelease.ReleasedBy, group.Name)
								}
								for _, handle := range entry.XMLReleasedBy.XMLHandle {
									// log.Println("XMLHandle: ", handle.XMLHandle)
									newRelease.ReleasedBy = append(newRelease.ReleasedBy, handle.XMLHandle)
								}

								// Linki dościągnięcia
								// Najpierw SIDy

								for _, link := range entry.DownloadLinks {
									newLink, _ := url.PathUnescape(link.Link)
									// log.Println("Download link: " + newLink)
									newRelease.DownloadLinks = append(newRelease.DownloadLinks, newLink)
								}

								//
								// Dodajemy
								//
								if len(newRelease.DownloadLinks) > 0 {
									// releases = append(releases, newRelease)
									DownloadRelease(newRelease)
									config.LastID = checkingID
									// Update konfiga (LastID) po każdym sprawdzeniu
									WriteConfig()
								}

							}
						}
					}
				} else {
					log.Println("csdb.dk communication error")
				}
			} else {
				log.Println("csdb.dk communication error")
				break
			}

			if checkingID < newestCSDbID {
				checkingID++
				config.LastID = checkingID
			} else {
				searching = false
			}

			// Odpoczynek
			time.Sleep(time.Millisecond * 200)
		}

	} else {
		log.Println("csdb.dk communication error")
	}
}

// DownloadRelease - Ściągnięcie pojedynczego release'u i zapisanie
// ================================================================================================
func DownloadRelease(release Release) {
	for _, downloadLink := range release.DownloadLinks {
		filename := filepath.Base(downloadLink)
		filename = filepath.Clean(filename)
		filename = strings.ReplaceAll(filename, "...", "")
		if release.ReleasedAt == "" {
			release.ReleasedAt = config.NoCompoDirectory
		}

		var groups string

		if len(release.ReleasedBy) > 0 {
			for i, group := range release.ReleasedBy {
				if i == len(release.ReleasedBy)-1 {
					groups += group
				} else {
					groups += group + " & "
				}
			}
		}

		// log.Println("Grupy: " + groups)

		var dir string

		if len(release.ReleasedBy) > 0 {
			dir = cacheDir + sep + release.ReleasedAt + sep + release.ReleaseType + sep + groups + sep + release.ReleaseName
		} else if len(release.Credits) > 0 {
			dir = cacheDir + sep + release.ReleasedAt + sep + release.ReleaseType + sep + groups + sep + release.ReleaseName
		} else {
			dir = cacheDir + sep + release.ReleasedAt + sep + release.ReleaseType + sep + "unknown" + sep + release.ReleaseName
		}

		dir = filepath.Clean(dir)
		dir = strings.ReplaceAll(dir, "...", "")

		if !fileExists(dir + sep + filename) {
			DownloadFile(dir, filename, downloadLink)
		}
	}
}

// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]
// ------------------------------------------------------------------------------------------------
// MAIN()
// ------------------------------------------------------------------------------------------------
// [][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]

func main() {

	gobackID := flag.Int("goback", 0, "How many IDs go back for updates -> change of config.LastID")
	startingID := flag.Int("start", 0, "Force ID number to start from -> change of config.LastID")
	date := flag.String("date", "", "Download only releases newer then date in form YYYY-MM-DD -> change of config.Date")
	looping := flag.Bool("loop", false, "Set to 'true' if you want to loop the program (default 'false')")
	allTypes := flag.Bool("all", false, "Set to 'true' if you want to ignore config.Types and download all types of releases (default 'false')")

	flag.Parse()

	//
	// Logowanie do pliku
	//
	logFileApp, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	ErrCheck(err)
	log.SetOutput(io.MultiWriter(os.Stdout, logFileApp))

	// Info powitalne
	//
	log.Println("==========================================")
	log.Println("=======         APP START         ========")
	log.Println("==========================================")

	sep = string(os.PathSeparator)

	//
	// Odczyt Configa
	//
	if fileExists("csdb-downloader.json") {
		ReadConfig()
	} else {
		config.DownloadDirectory = "csdb"
		config.NoCompoDirectory = "out_of_compo"
		config.LastID = 0
		config.Types = []string{"C64 Music", "C64 Graphics", "C64 Demo", "C64 One-File Demo", "C64 Intro", "C64 4K Intro", "C64 Crack Intro", "C64 Music Collection", "C64 Graphics Collection", "C64 Diskmag", "C64 Charts", "C64 Invitation", "C64 1K Intro", "C64 Fake Demo"}
	}

	cacheDir = config.DownloadDirectory
	log.Println("Download directory: " + cacheDir)

	// Czy podalismy datę?
	if *date != "" {
		config.Date = *date
	} else {
		*date = config.Date
	}

	WriteConfig()

	// Wykonanie pierwszy raz
	CSDBPrepareData(*gobackID, *startingID, *date, *allTypes)
	WriteConfig()

	// Start pętli
	for *looping {
		log.Println("Sleeping for minute...")
		time.Sleep(time.Minute)
		CSDBPrepareData(*gobackID, *startingID, *date, *allTypes)
		WriteConfig()
	}

	log.Println("==========================================")
	log.Println("=======          APP END.         ========")
	log.Println("==========================================")
}
