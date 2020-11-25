server {
    listen              443 ssl;
    server_name         {{.Domain}};
    ssl_certificate     conf.d/key/{{.Domain}}.crt;
    ssl_certificate_key conf.d/key/{{.Domain}}.key;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    if ( $host ~* {{.Domain}}) {
    	rewrite ^/(.*) {{.Url}} redirect;
    }

    location / {
        root   /usr/share/nginx/html;
        index  index.html index.htm;
    }

    location = /50x.html {
        root   /usr/share/nginx/html;
    }
}
