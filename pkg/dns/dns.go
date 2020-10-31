package dns

import (
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
)

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.AgentAcl{},
		&resource.AgentView{},
		&resource.AgentZone{},
		&resource.AgentForwardZone{},
		&resource.AgentRr{},
		&resource.AgentRedirection{},
		&resource.AgentDnsGlobalConfig{},
		&resource.AgentUrlRedirect{},
	}
}
