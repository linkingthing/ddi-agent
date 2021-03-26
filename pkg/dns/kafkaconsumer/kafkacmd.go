package kafkaconsumer

const (
	DNSTopic = "dns"

	StartDNS = "start_dns"
	StopDNS  = "stop_dns"

	CreateACL      = "create_acl"
	UpdateACL      = "update_acl"
	DeleteACL      = "delete_acl"
	BatchCreateACL = "batchcreate_acl"

	CreateView = "create_view"
	UpdateView = "update_view"
	DeleteView = "delete_view"

	CreateAuthZone        = "create_authzone"
	UpdateAuthZone        = "update_authzone"
	DeleteAuthZone        = "delete_authzone"
	CreateAuthZoneAuthRRs = "create_authzoneauthrrs"
	UpdateAuthZoneAXFR    = "update_authzoneaxfr"
	UpdateAuthZoneIXFR    = "update_authzoneixfr"

	CreateForwardZone = "create_forwardzone"
	UpdateForwardZone = "update_forwardzone"
	DeleteForwardZone = "delete_forwardzone"
	FlushForwardZone  = "flush_forwardzone"

	CreateAuthRR       = "create_authrr"
	UpdateAuthRR       = "update_authrr"
	DeleteAuthRR       = "delete_authrr"
	BatchCreateAuthRRs = "batchcreate_authrr"

	CreateRedirection = "create_redirection"
	UpdateRedirection = "update_redirection"
	DeleteRedirection = "delete_redirection"

	CreateNginxProxy = "create_nginxproxy"
	UpdateNginxProxy = "update_nginxproxy"
	DeleteNginxProxy = "delete_nginxproxy"

	UpdateGlobalConfig = "update_dnsglobalconfig"

	UploadLog = "upload_dnslog"
)
