This simple tool allows you to be notified as soon as an osu! accounts gets unbanned by automatically checking every 5 minutes.

Usage
============
If you're on windows, you can grab the binaries from the releases section and just fire the software up.

If you're on linux, I haven't had the chance to fire up my linux box and build the binaries just yet so you'll have to take a look at "How to compile".

Simply type the username or id of the desired player in the textbox and the tool will automatically start checking for unban every 5 minutes.

If you want a pop-up to notify you when the user is unbanned, click "Show pop-up on unban" so that it's highlighted.


How to compile
============
Make sure that you have git, mercurial and go installed and run the following commands to acquire all of the requires libraries.

	go get code.google.com/p/freetype-go/freetype
	go get github.com/go-gl/gl/v2.1/gl
	go get github.com/go-gl/glfw/v3.1/glfw
	go get github.com/google/gxui

Now grab the source code of the tool itself and install it:

	go get github.com/Francesco149/osu-unban-checker
	go install github.com/Francesco149/osu-unban-checker
	
You will find your binaries in $GOPATH/bin.