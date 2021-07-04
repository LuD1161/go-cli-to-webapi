# CLI TO WEB API

- Pass in any command to be run as a job


## ToDo
- [x] Add logging
- [x] Setup binaries inside the image 
- [ ] Create a process around it - Makefile - Add a config of tools that the image contains or can run. The image should pull these binaries while building and put those in $PATH
- [x] Deploy on k8s
- [ ] k8s ready - Add `nodename` or `ip` as unique `worker` string
- [ ] Pass other file inputs - like `nuclei -l file.txt`
- [ ] Add tests