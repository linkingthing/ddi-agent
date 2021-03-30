include "{{$.ConfigPath}}/cmcc_v4.conf";
include "{{$.ConfigPath}}/ctcc_v4.conf";
include "{{$.ConfigPath}}/cucc_v4.conf";
include "{{$.ConfigPath}}/cmcc_v6.conf";
include "{{$.ConfigPath}}/ctcc_v6.conf";
include "{{$.ConfigPath}}/cucc_v6.conf";

{{range $k, $acl := .Acls}}
acl "{{$acl.Name}}"{ {{range $k, $ip := $acl.Ips}}
{{$ip}};{{end}}
};{{end}}
