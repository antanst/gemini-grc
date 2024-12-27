# gemini-grc

A Gemini crawler.

URLs to visit as well as data from visited URLs are stored as "snapshots" in the database.
This makes it easily extendable as a "wayback machine" of Gemini.

## Done
- [x] Concurrent downloading with workers
- [x] Concurrent connection limit per host
- [x] URL Blacklist
- [x] Save image/* and text/* files
- [x] Configuration via environment variables
- [x] Storing snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation
- [x] Follow robots.txt, see gemini://geminiprotocol.net/docs/companion/robots.gmi
- [x] Handle redirects (3X status codes)
- [x] Better URL normalization

## TODO
- [ ] Add snapshot hash and support snapshot history
- [ ] Add web interface
- [ ] Provide a TLS cert for sites that require it, like Astrobotany

## TODO with lower priority
- [ ] Gopher
- [ ] Scroll gemini://auragem.letz.dev/devlog/20240316.gmi
- [ ] Spartan
- [ ] Nex
- [ ] SuperTXT https://supertxt.net/00-intro.html
