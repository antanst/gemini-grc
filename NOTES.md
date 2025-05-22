# Notes

Avoiding endless loops while crawling

- Make sure we follow robots.txt
- Announce our own agent so people can block us in their robots.txt
- Put a limit on number of pages per host, and notify on limit reach.
- Put a limit on the number of redirects (not needed?)

Heuristics:

- Do _not_ parse links from pages that have '/git/' or '/cgi/' or '/cgi-bin/' in their URLs.
- Have a list of "whitelisted" hosts/urls that we visit in regular intervals.
