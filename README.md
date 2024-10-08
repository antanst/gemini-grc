# gemini-grc

A Gemini crawler.

## TODO
- [ ] Save image/* and text/* files
- [ ] Wide events logging
- [ ] Handle URLs that need presentation of a TLS cert? Like astrobotany
  + [ ] Probably have a common "grc" cert for all
- [ ] Proper input and response validations:
  + [ ] When making a request, the URI MUST NOT exceed 1024 bytes
  + [ ] Response headers MUST be UTF-8 encoded text and MUST NOT begin with the Byte Order Mark U+FEFF.
- [ ] Proper handling of all response codes
- [ ] Proper validation (or logging) of invalid/expired TLS certs?
- [ ] Subscribe to gemini pages? gemini://geminiprotocol.net/docs/companion/
- [ ] Follow robots.txt gemini://geminiprotocol.net/docs/companion/

## TODO later
- [ ] Add other protocols
  + [ ] Scroll gemini://auragem.letz.dev/devlog/20240316.gmi
  + [ ] Spartan
  + [ ] Nex
  + [ ] SuperTXT https://supertxt.net/00-intro.html
