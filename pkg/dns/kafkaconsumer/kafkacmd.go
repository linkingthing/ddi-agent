package kafkaconsumer

const (
	DNSTopic = "dns"

	StartDNS = "start_dns"
	StopDNS  = "stop_dns"

	CreateACL = "create_acl"
	UpdateACL = "update_acl"
	DeleteACL = "delete_acl"

	CreateView = "create_view"
	UpdateView = "update_view"
	DeleteView = "delete_view"

	CreateZone = "create_zone"
	UpdateZone = "update_zone"
	DeleteZone = "delete_zone"

	CreateForwardZone = "create_forwardzone"
	UpdateForwardZone = "update_forwardzone"
	DeleteForwardZone = "delete_forwardzone"
	FlushForwardZone  = "flush_forwardzone"

	CreateRR = "create_rr"
	UpdateRR = "update_rr"
	DeleteRR = "delete_rr"

	CreateRedirection = "create_redirection"
	UpdateRedirection = "update_redirection"
	DeleteRedirection = "delete_redirection"

	CreateUrlRedirect = "create_urlredirect"
	UpdateUrlRedirect = "update_urlredirect"
	DeleteUrlRedirect = "delete_urlredirect"

	UpdateGlobalConfig = "update_dnsglobalconfig"

	UploadLog = "upload_dnslog"
)
