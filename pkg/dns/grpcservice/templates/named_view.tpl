{{range $k, $view := .Views}}
key key{{$view.Name}} {
    algorithm hmac-md5;
    secret "{{$view.Key}}";
};
{{end}}

{{range $k, $view := .Views}}
view "{{$view.Name}}" {
	match-clients {
	{{range $kk, $acl := $view.ACLs}}{{$acl.Name}};{{end}}
	key key{{$view.Name}};
	};
	allow-update {key key{{$view.Name}};};{{range $i, $zone := $view.Zones}}
	zone "{{$zone.Name}}" { type forward; forward {{$zone.ForwardType}}; forwarders { {{range $ii,$ip := $zone.IPs}}{{$ip}}; {{end}}}; };{{end}}{{range $k, $dns64:= .DNS64s}}
        dns64 {{$dns64.Prefix}} {
        clients { {{$dns64.ClientACLName}}; };
        mapped { {{$dns64.AAddressACLName}}; };
        suffix ::;
        };{{end}}{{if $view.Redirect}}
	zone "." {
        type redirect;
        file "redirection/redirect_{{$view.Name}}";
        };{{end}}{{if $view.RPZ}}
	response-policy { zone "rpz" policy given; } max-policy-ttl 86400 qname-wait-recurse no ;
        zone "rpz" {type master; file "redirection/rpz_{{$view.Name}}"; allow-query {any;}; };{{end}}
};{{end}}
