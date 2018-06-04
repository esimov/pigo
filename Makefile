all: 
	@./build.sh
clean:
	@rm -f pigo
install: all
	@cp pigo /usr/local/bin
uninstall: 
	@rm -f /usr/local/bin/pigo
package:
	@NOCOPY=1 ./build.sh package