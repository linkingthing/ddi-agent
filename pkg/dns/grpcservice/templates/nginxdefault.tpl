server {
    listen       80;
    listen  [::]:80;
    {{range $k,$v:=.URLRedirects}}
    if ( $host ~* {{$v.Domain}}) {
    rewrite ^/(.*) {{$v.URL}} redirect;
    }
    {{end}}
    location / {
        root   /usr/share/nginx/html;
        index  index.html index.htm;
    }

    location = /50x.html {
        root   /usr/share/nginx/html;
    }
}
