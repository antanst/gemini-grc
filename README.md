# gemini-grc

A crawler for the [Gemini](https://en.wikipedia.org/wiki/Gemini_(protocol)) network. Easily extendable as a "wayback machine" of Gemini.

## Features done
- [x] URL normalization
- [x] Handle redirects (3X status codes)
- [x] Follow robots.txt, see gemini://geminiprotocol.net/docs/companion/robots.gmi
- [x] Save image/* and text/* files
- [x] Concurrent downloading with workers
- [x] Connection limit per host
- [x] URL Blacklist
- [x] Configuration via environment variables
- [x] Storing snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation

## TODO
- [ ] Add snapshot history
- [ ] Add a web interface
- [ ] Provide to servers a TLS cert for sites that require it, like Astrobotany

## TODO (lower priority)
- [ ] Gopher
- [ ] Scroll gemini://auragem.letz.dev/devlog/20240316.gmi
- [ ] Spartan
- [ ] Nex
- [ ] SuperTXT https://supertxt.net/00-intro.html
