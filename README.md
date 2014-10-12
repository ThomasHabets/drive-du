Drive-DU
Copyright(c) Thomas Habets <thomas@habets.se> 2014

Running at: https://drive-du.appspot.com/
Also contains cmdline tools for working with Google Drive.


Command line tools
==================
To use these, first you need to create a project in
[the Google Developers Console](https://console.developers.google.com).

Create a project, then create credentials for a "native
application". Use the given ClientID and ClientSecret you're assigned.

du
--
Run ```./du -config=du.json -configure``` and follow the instructions, then
```./du -config=du.json 0Bn_HTPNhtnhTNHTNUHNhtn

find
----
Same as above, but with the ```find``` binary.

chown
-----
This doesn't actually change the ownership of files, since outside of
Google Enterprise something-or-other, that's not possible. So instead
it copies the files from one folder (```-src```) to another
(```-dst```). The owner of the files in ```-dst``` will be the oauth`ed
user, and the ```-src``` folder must be readable by this user.
