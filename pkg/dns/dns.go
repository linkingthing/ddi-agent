package dns

import (
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
)

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.AgentAcl{},
		&resource.AgentView{},
		&resource.AgentAuthZone{},
		&resource.AgentForwardZone{},
		&resource.AgentAuthRr{},
		&resource.AgentRedirection{},
		&resource.AgentDnsGlobalConfig{},
		&resource.AgentUrlRedirect{},
	}
}
