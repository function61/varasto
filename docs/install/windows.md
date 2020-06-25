Windows
=======

Follow same instructions as for [Linux (manual installation)](linux-manual.md), but
there's no autostart yet (the `server install` thing), so you have to just run the .exe
file directly from command line.


Supported Windows versions
--------------------------

Only Windows 10 works, because it introduced some
[features that we need](https://devblogs.microsoft.com/commandline/af_unix-comes-to-windows/).


Future of our Windows support
-----------------------------

In the future I think we should research targeting
[Windows Subsystem for Linux](https://en.wikipedia.org/wiki/Windows_Subsystem_for_Linux)
(present since Win10) via Docker to have less moving parts.

