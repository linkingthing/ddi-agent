options {
	directory "{{.ConfigPath}}";
	pid-file "named.pid";
	allow-new-zones yes;
	allow-query {any;};
	{{if .IsDnssecOpen}}dnssec-enable yes;{{else}}dnssec-enable no;{{end}}
	dnssec-validation no;
	{{if .IsLogOpen}}querylog yes;{{else}}querylog no;{{end}}{{if .IPBlackHole}}
	BlackHole{ {{range $k,$v := .IPBlackHole.ACLNames}}{{$v}}; {{end}}};{{end}}{{if .Concu}}
	recursive-clients {{.Concu.RecursiveClients}};
	fetches-per-zone {{.Concu.FetchesPerZone}};{{end}}{{if .SortList}}
	sortlist{ {{range $k, $s := .SortList}}{{$s}};{{end}} };{{end}}
};
statistics-channels {
     inet 0.0.0.0 port 58082;
};
{{range $k, $view := .Views}}
key key{{$view.Name}} {
    algorithm hmac-md5;
    secret "{{$view.Key}}";
};
{{end}}

{{if .IsLogOpen}}logging{
	channel query_log{
	buffered true;
	file "query.log" versions 5 size 200m;
	print-time yes;
	print-category yes;
        severity dynamic;
	};
	category queries{
	query_log;
	};
};{{end}}

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
        exclude { {{$dns64.Prefix}}; };
        suffix ::;
        };{{end}}{{if $view.Redirect}}
	zone "." {
        type redirect;
        file "redirection/redirect_{{$view.Name}}";
        };{{end}}{{if $view.RPZ}}
	response-policy { zone "rpz" policy given; } max-policy-ttl 86400 qname-wait-recurse no ;
        zone "rpz" {type master; file "redirection/rpz_{{$view.Name}}"; allow-query {any;}; };{{end}}
};{{end}}

key "rndc-key" {
	algorithm hmac-sha256;
	secret "4WqnJgCtpG8dPHDCBjwyQKtOzAPgiS+Iah5KN4xeq/U=";
};
controls {
        inet 127.0.0.1 port 953
        allow { 127.0.0.1; } keys { "rndc-key"; };
};

include "{{$.ConfigPath}}/cmcc.conf";
include "{{$.ConfigPath}}/ctcc.conf";
include "{{$.ConfigPath}}/cucc.conf";
{{range $k, $acl := .ACLNames}}
include "{{$.ConfigPath}}/{{$acl}}.acl";{{end}}

