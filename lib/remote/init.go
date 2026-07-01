package remote

import (
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

func InitScanners(remote *Library) {
	numverifySupplier := suppliers.NewNumverifySupplier()
	ovhSupplier := suppliers.NewOVHSupplier()
	twilioSupplier := suppliers.NewTwilioSupplier()
	dehashedSupplier := suppliers.NewDehashedSupplier()
	hlrSupplier := suppliers.NewHLRSupplier()
	ipqsSupplier := suppliers.NewIPQSSupplier()
	nanpaSupplier := suppliers.NewNANPASupplier()
	serpapiSupplier := suppliers.NewSerpAPISupplier()

	remote.AddScanner(NewLocalScanner())
	remote.AddScanner(NewNumverifyScanner(numverifySupplier))
	remote.AddScanner(NewGoogleSearchScanner())
	remote.AddScanner(NewSearXNGScanner(nil))
	remote.AddScanner(NewOVHScanner(ovhSupplier))
	remote.AddScanner(NewTwilioScanner(twilioSupplier))
	remote.AddScanner(NewBreachScanner(dehashedSupplier))
	remote.AddScanner(NewHLRScanner(hlrSupplier))
	remote.AddScanner(NewIPQualityScoreScanner(ipqsSupplier))
	remote.AddScanner(NewValidationScanner(suppliers.NewVeriphoneProvider()))
	remote.AddScanner(NewValidationScanner(suppliers.NewAbstractProvider()))
	remote.AddScanner(NewValidationScanner(suppliers.NewNumlookupAPIProvider()))
	remote.AddScanner(NewNANPAScanner(nanpaSupplier))
	remote.AddScanner(NewSerpAPIScanner(serpapiSupplier))

	remote.LoadPlugins()
}
