acl "{{.Name}}"{ {{range $k, $ip := .Ips}}
{{$ip}};{{end}}
};
