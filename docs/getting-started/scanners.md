# Scanners

PhoneInfoga provide several scanners to extract as much information as possible from a given phone number. Those scanners may require authentication, so they're automatically skipped when no authentication credentials are found.

## Configuration

Note that all scanners use environment variables for configuration values. You can define an environment variable inline or put them in a file called `.env` in the current directory. The tool will parse it automatically. To specify another filename, use the flag `--env-file`.

**Example**

```shell
# .env.local
NUMVERIFY_API_KEY="value"
```

```shell
phoneinfoga scan -n +4176418xxxx --env-file=.env.local
```

### Scanner options

When using the **REST API**, you can also specify those values on a per-request basis. Each scanner supports its own options, see below. For details on how to specify those options, see [API docs](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/sundowndev/phoneinfoga/master/web/docs/swagger.yaml#/Numbers/RunScanner). For readability and simplicity, options are named exactly like their environment variable equivalent.

!!! warning
    Scanner options will override environment variables for the current request.

## Building your own scanner

PhoneInfoga can now be extended with plugins! You can build your own scanner and PhoneInfoga will use it to scan the given phone number.

```shell
$ phoneinfoga scan -n +4176418xxxx --plugin ./custom_scanner.so
```

!!! info
    Plugins are written with the [Go programming language](https://golang.org/). To get started, [see this example plugin](https://github.com/sundowndev/phoneinfoga/tree/master/examples/plugin).

## Local

The local scan is probably the simplest scan of PhoneInfoga. By default, the tool statically parse the phone number and convert it to several formats, it also tries to recognize the country and the carrier. This information are passed to all scanners in order to provide further analysis. The local scanner simply return those information to the end user, so they can exploit it as well.

??? info "Configuration"

    There is no configuration required for this scanner.

??? example "Output example"

    ```shell
    $ phoneinfoga scan -n +4176418xxxx
    
    Results for local
    Raw local: 076418xxxx
    Local: 076 418 xx xx
    E164: +4176418xxxx
    International: 4176418xxxx
    Country: CH
    ```

## Numverify

Numverify provide standard but useful information such as country code, location, line type and carrier. This scanners requires an API-key which you can get on their website after creating an account. You can use a free API key as long as you don't exceed the monthly quota. **This is an [apilayer](https://apilayer.com/marketplace/number_verification-api) key, not numverify itself.**

[Read documentation](https://apilayer.com/marketplace/number_verification-api#details-tab)

??? info "Configuration"

    1. Go to the [Api layer website](https://apilayer.com/) and create an account
    2. Go to "Number Verification API" in the marketplace, click on "Subscribe for free", then choose whatever plan you want
    3. Copy the new API token and use it as an environment variable

    | Environment variable |   Option   | Default | Description                                          |
    |----------------------|------------|---------|-------------------------------------------------------|
    | NUMVERIFY_API_KEY    |   NUMVERIFY_API_KEY  |         | API key to authenticate to the Numverify API.        |

??? example "Output example"

    ```shell
    $ NUMVERIFY_API_KEY=<key> phoneinfoga scan -n +4176418xxxx
    
    Results for numverify
    Valid: true
    Number: 4176418xxxx
    Local format: 076418xxxx
    International format: +4176418xxxx
    Country prefix: +41
    Country code: CH
    Country name: Switzerland (Confederation of)
    Carrier: Sunrise Communications AG
    Line type: mobile
    ```

## Googlesearch

Googlesearch uses the Google search engine and [Google Dorks](https://en.wikipedia.org/wiki/Google_hacking) to search phone number's footprints everywhere on the web. It allows you to search for scam reports, social media profiles, documents and more. **This scanner does only one thing:** generating several Google search links from a given phone number. You then have to manually open them in your browser to see results. So the tool may generate links that do not return any result. This is a design choice we made to avoid technical limitation around [Google scraping](https://en.wikipedia.org/wiki/Search_engine_scraping).

You can however, use this scanner through the REST API in addition with another tool to fetch the result automatically. If you wish to retrieve results automatically, see the [SerpAPI scanner](#serpapi) instead.

??? info "Configuration"

    There is no configuration required for this scanner.

??? example "Output example"

    ```shell
    $ phoneinfoga scan -n +4176418xxxx
    
    Results for googlesearch
    Social media:
        URL: https://www.google.com/search?q=site%3Afacebook.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Atwitter.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Alinkedin.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Ainstagram.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Avk.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    Disposable providers:
        URL: https://www.google.com/search?q=site%3Ahs3x.com+intext%3A%224176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceive-sms-now.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asmslisten.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asmsnumbersonline.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Afreesmscode.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Acatchsms.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asmstibo.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asmsreceiving.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Agetfreesmsnumber.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asellaite.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceive-sms-online.info+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceivesmsonline.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceive-a-sms.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asms-receive.net+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceivefreesms.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceive-sms.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceivetxt.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Afreephonenum.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Afreesmsverification.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Areceive-sms-online.com+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Asmslive.co+intext%3A%224176418xxxx%22+OR+intext%3A%22076418xxxx%22
    Reputation:
        URL: https://www.google.com/search?q=site%3Awhosenumber.info+intext%3A%22%2B4176418xxxx%22+intitle%3A%22who+called%22
    
        URL: https://www.google.com/search?q=intitle%3A%22Phone+Fraud%22+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Afindwhocallsme.com+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%224176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Ayellowpages.ca+intext%3A%22%2B4176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Aphonenumbers.ie+intext%3A%22%2B4176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Awho-calledme.com+intext%3A%22%2B4176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Ausphonesearch.net+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Awhocalled.us+inurl%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Aquinumero.info+intext%3A%22076418xxxx%22+OR+intext%3A%224176418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Auk.popularphotolook.com+inurl%3A%22076418xxxx%22
    Individuals:
        URL: https://www.google.com/search?q=site%3Anuminfo.net+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Async.me+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Awhocallsyou.de+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Apastebin.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Awhycall.me+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Alocatefamily.com+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    
        URL: https://www.google.com/search?q=site%3Aspytox.com+intext%3A%22076418xxxx%22
    General:
        URL: https://www.google.com/search?q=intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22+OR+intext%3A%22076+418+xx+xx%22
    
        URL: https://www.google.com/search?q=%28ext%3Adoc+OR+ext%3Adocx+OR+ext%3Aodt+OR+ext%3Apdf+OR+ext%3Artf+OR+ext%3Asxw+OR+ext%3Apsw+OR+ext%3Appt+OR+ext%3Apptx+OR+ext%3Apps+OR+ext%3Acsv+OR+ext%3Atxt+OR+ext%3Axls%29+intext%3A%224176418xxxx%22+OR+intext%3A%22%2B4176418xxxx%22+OR+intext%3A%22076418xxxx%22
    ```

## OVH

OVH, besides being a web and cloud hosting company, is a telecom provider with several VoIP numbers in Europe. Thanks to their API-key free REST API, we are able to tell if a number is owned by OVH Telecom or not.

??? info "Configuration"

    There is no configuration required for this scanner.

??? example "Output example"

    ```shell
    $ phoneinfoga scan -n +3336517xxxx

    Results for ovh
    Found: true
    Number range: 036517xxxx
    City: Abbeville
    ```

## HLR

The HLR (Home Location Register) scanner reports the *live* network state of a mobile number: whether it is currently reachable, the carrier serving it after any porting, roaming status and line type. Unlike range/validation lookups it reflects the number's state right now, so providers bill per lookup. It is provider-agnostic: point it at any HLR/MNP provider (Abstract HLR, IPQS HLR, apilayer, …) via `HLR_API_URL`.

The scanner only runs for **mobile** numbers (re-derived with libphonenumber) and is skipped cleanly until both an API key and a real `HLR_API_URL` are configured.

??? info "Configuration"

    | Environment variable | Option        | Default | Description                                                              |
    |----------------------|---------------|---------|--------------------------------------------------------------------------|
    | HLR_API_KEY          | HLR_API_KEY   |         | Provider API key.                                                        |
    | HLR_API_URL          | HLR_API_URL   |         | Provider lookup endpoint. Required; the scanner skips while it is empty. |

??? example "Output example"

    ```shell
    $ HLR_API_KEY=<key> HLR_API_URL=https://hlr.example.com/lookup phoneinfoga scan -n +33612345678

    Results for hlr
    Active: true
    Status: active
    Current carrier: Orange
    Line type: mobile
    Country: FR
    ```

## IPQualityScore

IPQualityScore scores a phone number for fraud and abuse signals (fraud score, recent abuse, VOIP/prepaid/risky flags) alongside carrier, line type and coarse geography. Requires an API key obtained from your IPQualityScore account.

[Read documentation](https://www.ipqualityscore.com/documentation/phone-number-validation-api/overview)

??? info "Configuration"

    | Environment variable | Option       | Default | Description                                            |
    |----------------------|--------------|---------|--------------------------------------------------------|
    | IPQS_API_KEY         | IPQS_API_KEY |         | API key to authenticate to the IPQualityScore API.     |

??? example "Output example"

    ```shell
    $ IPQS_API_KEY=<key> phoneinfoga scan -n +14159929960

    Results for ipqualityscore
    Valid: true
    Active: true
    Fraud score: 0
    Recent abuse: false
    VOIP: false
    Prepaid: false
    Risky: false
    Line type: Wireless
    Carrier: T-Mobile USA
    Country: US
    ```

## Validation (Veriphone, Abstract, Numlookup)

These are drop-in alternatives to Numverify, each providing standard validation data (validity, formats, country, carrier, line type). They share one scanner implementation but register as three independent scanners — configure only the providers you use; each runs when its own key is present.

| Scanner name   | Provider     | Documentation                                            |
|----------------|--------------|----------------------------------------------------------|
| `veriphone`    | Veriphone    | <https://veriphone.io/docs>                              |
| `abstract`     | Abstract     | <https://www.abstractapi.com/api/phone-validation-api>   |
| `numlookupapi` | NumlookupAPI | <https://numlookupapi.com/docs>                          |

??? info "Configuration"

    | Environment variable    | Option                  | Default | Description                                  |
    |-------------------------|-------------------------|---------|----------------------------------------------|
    | VERIPHONE_API_KEY       | VERIPHONE_API_KEY       |         | API key for the Veriphone validator.         |
    | ABSTRACT_PHONE_API_KEY  | ABSTRACT_PHONE_API_KEY  |         | API key for the Abstract phone validator.    |
    | NUMLOOKUPAPI_API_KEY    | NUMLOOKUPAPI_API_KEY    |         | API key for the NumlookupAPI validator.      |

??? example "Output example"

    ```shell
    $ VERIPHONE_API_KEY=<key> phoneinfoga scan -n +14159929960

    Results for veriphone
    Provider: veriphone
    Valid: true
    International format: +1 415-992-9960
    Country name: United States
    Carrier: Bandwidth.com
    Line type: MOBILE
    ```

## NANPA

The NANPA scanner resolves North American (+1) numbers to their NPA-NXX allocation: rate center, state, carrier-of-record (OCN) and LATA. It generalizes the range concept the `ovh` scanner provides for parts of Europe to the North American Numbering Plan, behind a configurable endpoint.

It runs only for **+1** numbers and is skipped cleanly while `NANPA_API_URL` is empty. `NANPA_API_KEY` is optional and only sent when your provider requires it.

??? info "Configuration"

    | Environment variable | Option         | Default | Description                                                            |
    |----------------------|----------------|---------|------------------------------------------------------------------------|
    | NANPA_API_URL        | NANPA_API_URL  |         | NPA-NXX lookup endpoint. Required; the scanner skips while it is empty. |
    | NANPA_API_KEY        | NANPA_API_KEY  |         | Optional API key, only if the provider requires authentication.        |

??? example "Output example"

    ```shell
    $ NANPA_API_URL=https://npa-nxx.example.com/v1 phoneinfoga scan -n +12025550123

    Results for nanpa
    Found: true
    Area code: 202
    Exchange: 555
    Rate center: WASHINGTON DC
    State: DC
    OCN: 0001
    Carrier of record: Example Telco
    LATA: 236
    ```

## SerpAPI

SerpAPI runs the same phone-number footprint dorks as the `googlesearch` scanner, but executes them through [SerpAPI](https://serpapi.com/) and returns parsed organic results automatically (rather than links you open manually). Useful for programmatic footprinting across one or more search engines.

??? info "Configuration"

    | Environment variable | Option              | Default  | Description                                                       |
    |----------------------|---------------------|----------|-------------------------------------------------------------------|
    | SERPAPI_KEY          | SERPAPI_KEY         |          | API key to authenticate to SerpAPI.                              |
    | SERPAPI_ENGINES      | SERPAPI_ENGINES     | google   | Comma-separated SerpAPI engines to query.                        |
    | SERPAPI_MAX_RESULTS  | SERPAPI_MAX_RESULTS | 3        | Maximum organic results kept per dork (0 means unlimited).       |

??? example "Output example"

    ```shell
    $ SERPAPI_KEY=<key> phoneinfoga scan -n +14159929960

    Results for serpapi
    Reputation:
        Query: site:whocalled.us inurl:"4159929960"
        Engine: google
        Result count: 2
        Results:
            Title: Who called from 415-992-9960?
            URL: https://whocalled.us/...
    ```
