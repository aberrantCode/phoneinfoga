package remote

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
)

const Googlesearch = "googlesearch"

type googlesearchScanner struct{}

// GoogleSearchDork is the common format for dork requests
type GoogleSearchDork struct {
	Number string `json:"number" console:"-"`
	Dork   string `json:"dork" console:"-"`
	URL    string `json:"url" console:"URL"`
}

// GoogleSearchResponse is the output of Google search scanner.
// It contains all dorks created ordered by types.
type GoogleSearchResponse struct {
	SocialMedia         []*GoogleSearchDork `json:"social_media" console:"Social media,omitempty"`
	DisposableProviders []*GoogleSearchDork `json:"disposable_providers" console:"Disposable providers,omitempty"`
	Reputation          []*GoogleSearchDork `json:"reputation" console:"Reputation,omitempty"`
	Individuals         []*GoogleSearchDork `json:"individuals" console:"Individuals,omitempty"`
	General             []*GoogleSearchDork `json:"general" console:"General,omitempty"`
}

func NewGoogleSearchScanner() Scanner {
	return &googlesearchScanner{}
}

func (s *googlesearchScanner) Name() string {
	return Googlesearch
}

func (s *googlesearchScanner) Description() string {
	return "Generate several Google dork requests for a given phone number."
}

func (s *googlesearchScanner) DryRun(_ number.Number, _ ScannerOptions) error {
	return nil
}

func (s *googlesearchScanner) Run(n number.Number, _ ScannerOptions) (interface{}, error) {
	res := GoogleSearchResponse{
		SocialMedia:         getSocialMediaDorks(n),
		DisposableProviders: getDisposableProvidersDorks(n),
		Reputation:          getReputationDorks(n),
		Individuals:         getIndividualsDorks(n),
		General:             getGeneralDorks(n),
	}

	return res, nil
}

func getDisposableProvidersDorks(number number.Number) (results []*GoogleSearchDork) {
	return makeDorks(number, []string{
		siteQuery("receive-sms-online.com", number),
		siteQuery("sms24.me", number),
		siteQuery("quackr.io", number),
		siteQuery("receive-smss.com", number),
		siteQuery("freephonenum.com", number),
		siteQuery("receive-sms.cc", number),
	})
}

func getIndividualsDorks(number number.Number) (results []*GoogleSearchDork) {
	variants := numberVariants(number)
	return makeDorks(number, []string{
		fmt.Sprintf("(%s) (\"contact\" OR \"phone\" OR \"mobile\" OR \"tel\")", variants),
		fmt.Sprintf("(%s) (\"address\" OR \"directory\" OR \"profile\")", variants),
		fmt.Sprintf("(%s) (\"resume\" OR \"cv\" OR \"vcard\")", variants),
		siteQuery("pastebin.com", number),
		siteQuery("github.com", number),
		siteQuery("keybase.io", number),
		siteQuery("locatefamily.com", number),
		siteQuery("sync.me", number),
	})
}

func getSocialMediaDorks(number number.Number) (results []*GoogleSearchDork) {
	return makeDorks(number, []string{
		siteQuery("facebook.com", number),
		siteQuery("x.com", number),
		siteQuery("twitter.com", number),
		siteQuery("linkedin.com", number),
		siteQuery("instagram.com", number),
		siteQuery("tiktok.com", number),
		siteQuery("reddit.com", number),
		siteQuery("vk.com", number),
	})
}

func getReputationDorks(number number.Number) (results []*GoogleSearchDork) {
	variants := numberVariants(number)
	return makeDorks(number, []string{
		fmt.Sprintf("(%s) (\"who called\" OR \"called me\" OR \"unknown caller\")", variants),
		fmt.Sprintf("(%s) (\"spam\" OR \"scam\" OR \"fraud\" OR \"robocall\")", variants),
		siteQuery("800notes.com", number),
		siteQuery("whocallsme.com", number),
		siteQuery("tellows.com", number),
		siteQuery("nomorobo.com", number),
		siteQuery("shouldianswer.com", number),
		siteQuery("truecaller.com", number),
	})
}

func getGeneralDorks(number number.Number) (results []*GoogleSearchDork) {
	variants := numberVariants(number)
	return makeDorks(number, []string{
		variants,
		fmt.Sprintf("(%s) (\"contact\" OR \"phone\" OR \"call\" OR \"text\")", variants),
		fmt.Sprintf("(%s) (filetype:pdf OR filetype:doc OR filetype:docx OR filetype:xls OR filetype:xlsx OR filetype:csv OR filetype:txt)", variants),
		fmt.Sprintf("(%s) (\"hours\" OR \"location\" OR \"appointment\" OR \"booking\")", variants),
		fmt.Sprintf("(%s) (\"WhatsApp\" OR \"Telegram\" OR \"Signal\")", variants),
	})
}

func makeDorks(number number.Number, queries []string) (results []*GoogleSearchDork) {
	for _, query := range queries {
		results = append(results, &GoogleSearchDork{
			Number: number.E164,
			Dork:   query,
			URL:    "https://www.google.com/search?q=" + url.QueryEscape(query),
		})
	}

	return results
}

func siteQuery(site string, number number.Number) string {
	return fmt.Sprintf("site:%s (%s)", site, numberVariants(number))
}

func numberVariants(number number.Number) string {
	var quoted []string
	seen := map[string]bool{}
	for _, variant := range []string{
		number.E164,
		number.International,
		number.RawLocal,
		number.Local,
		strings.ReplaceAll(number.Local, " ", "-"),
	} {
		if variant == "" || seen[variant] {
			continue
		}
		seen[variant] = true
		quoted = append(quoted, fmt.Sprintf("%q", variant))
	}

	return strings.Join(quoted, " OR ")
}
