acl "{{.Name}}"{ {{range $k, $ip := .IPs}}
{{$ip}};{{end}}
};
