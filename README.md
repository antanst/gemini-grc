# gemini-grc

A Gemini crawler.

## Done
- [x] Concurrent downloading with workers
- [x] Concurrent connection limit per host
- [x] URL Blacklist
- [x] Save image/* and text/* files
- [x] Configuration via environment variables
- [x] Storing snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation
- [x] Follow robots.txt

## TODO
- [ ] Take into account gemini://geminiprotocol.net/docs/companion/robots.gmi
- [ ] Proper handling of all response codes
  - [ ] Handle 3X redirects properly
- [ ] Handle URLs that need presentation of a TLS cert, like astrobotany
  + [ ] Probably have a common "grc" cert for all?
- [ ] Proper input and response validations:
  + [ ] When making a request, the URI MUST NOT exceed 1024 bytes
- [ ] Subscriptions to gemini pages? gemini://geminiprotocol.net/docs/companion/

## TODO for later
- [ ] Add other protocols
  + [ ] Gopher
  + [ ] Scroll gemini://auragem.letz.dev/devlog/20240316.gmi
  + [ ] Spartan
  + [ ] Nex
  + [ ] SuperTXT https://supertxt.net/00-intro.html
