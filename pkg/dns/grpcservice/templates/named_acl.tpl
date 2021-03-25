include "{{$.ConfigPath}}/cmcc.conf";
include "{{$.ConfigPath}}/ctcc.conf";
include "{{$.ConfigPath}}/cucc.conf";
include "{{$.ConfigPath}}/cmcc_v6.conf";
include "{{$.ConfigPath}}/ctcc_v6.conf";
include "{{$.ConfigPath}}/cucc_v6.conf";

{{range $k, $acl := .Acls}}
acl "{{$acl.Name}}"{ {{range $k, $ip := $acl.Ips}}
{{$ip}};{{end}}
};{{end}}
