all: 
	@./build.sh
clean:
	@rm -f triangle
install: all
	@cp triangle /usr/local/bin
uninstall: 
	@rm -f /usr/local/bin/triangle
package:
	@NOCOPY=1 ./build.sh package