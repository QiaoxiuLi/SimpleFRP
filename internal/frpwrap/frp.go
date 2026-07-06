package frpwrap

import frpversion "github.com/fatedier/frp/pkg/util/version"

func UpstreamFRPVersion() string {
	return frpversion.Full()
}
