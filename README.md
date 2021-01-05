# registry-acl-proxy-gitea

> Simple proxy to use Gitea authentication as docker registry access control list.

## TLDR

### Installation

#### Docker

```
docker run \
    --name rapg \
    -e "DEBUG=1" \
    -e "GITEA_HOST=https://git.my.tld" \
    -e "READ_ONLY_USERNAMES=paulo,pedro" \
    -p8787:8787 \
    pierrecle/registry-acl-proxy-gitea:latest
```

#### Docker compose

```
  registry-acl-proxy-gitea:
    image: pierrecle/registry-acl-proxy-gitea:latest
    container_name: registry-acl-proxy-gitea
    restart: unless-stopped
    ports:
      - 8787:8787
    volumes:
      - /usr/share/zoneinfo:/usr/share/zoneinfo:ro
      - /etc/localtime:/etc/localtime:ro
    environment:
      - DEBUG=1
      - GITEA_HOST=https://git.my.tld
      - READ_ONLY_USERNAMES=paulo,pedro
```

### Options

* `GITEA_HOST`: Gitea host (example: `http://git.home.tld`, default: `empty`)
* `DEBUG`: display debug informations and log every request (`[0|1]`, default: `0`)
* `ALLOW_ANONYMOUS_READ`: allow unauthenticated `GET` and `HEAD` requests (`[0|1]`, default: `0`)
* `READ_ONLY_USERNAMES`: list (comma separated) of Gitea usernames that can only perform `GET` and `HEAD` requests (default: `empty`)
* `REALM`: Realm if authentication is needed (default: `Registry authentication`)

### Nginx configuration

Put the following configuration in your registry proxy configuration (in nginx-proxy-manager in `Advanced` > `Custom Nginx Configuration`).

```
  location / {
    # Authorization
    auth_request          /_auth;
    add_header 'Docker-Distribution-Api-Version' 'registry/2.0' always;

    # Force SSL -> Nginx-Proxy-Manager only
    #include conf.d/include/force-ssl.conf;

    # Proxy! -> Nginx-Proxy-Manager only
    #include conf.d/include/proxy.conf;
  }

  location = /_auth {
    internal;
    proxy_pass http://[replace_rapg_host]:[replace_rapg_port];
    proxy_pass_request_body off;
    proxy_pass_request_headers on;
    proxy_set_header  Authorization $http_authorization;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Original-Remote-Addr $remote_addr;
    proxy_set_header X-Original-Host $host;
  }
```

## The problem

For my personnal use, I want a light stack to handle my personnal git repositories and repositories for few friends, and a docker registry to handle personnal docker images. Gitea and docker registry are light and easy to setup enough for my needs, but docker registry a custom authentication layer, unless you use basic auth. Thing is I don't want to handle users twice (in Gitea and docker registry).

## What registry-acl-proxy-gitea can do or not

* it __can__ limit access to the registry according to Gitea authentication (ie. only Gitea users with their password (or access token) can access registry)
* it __can__ allow read request for unauthenticated calls
* it __can__ prevent users to perform write operations in registry
* it __cannot__ rewrite `catalog` call (ie cannot filter the list of repositories according to user rights)
* it __forces__ push/delete images to user's repositories (ie `docker push .../<username>/<projectname>`)

Finnally, it's just an `nginx` middleware to handle auth using Gitea.

## Requirements

* `nginx` configured as proxy for `docker registry v2`
* `nginx` must have [`ngx_http_auth_request_module`](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html)

I use it with [`nginx-proxy-manager`](https://nginxproxymanager.com/)

## How it works

Big picture: `nginx` request the proxy with the given authentication information. The proxy request Gitea with the given credential. If Gitea request fail, user request will fail too.

### Example :
`registry-acl-proxy-gitea` is called `rapg` to enhance readability.

```
 user | --      GET /v2/_catalog     -> | nginx
nginx | --      GET Auth ...         -> | rapg
 rapg | --      GET /api/v1/user     -> | gitea
 rapg | <-          user data or 401 -- | gitea
nginx | <-  200 or 401 after proces. -- | rapg
 user | <- [registry] request or 401 -- | nginx
```