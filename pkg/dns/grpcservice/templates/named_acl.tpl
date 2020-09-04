include "{{$.ConfigPath}}/cmcc.conf";
include "{{$.ConfigPath}}/ctcc.conf";
include "{{$.ConfigPath}}/cucc.conf";

{{range $k, $acl := .Acls}}
acl "{{$acl.Name}}"{ {{range $k, $ip := $acl.Ips}}
{{$ip}};{{end}}
};{{end}}
