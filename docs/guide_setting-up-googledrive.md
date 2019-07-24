Setting up Google Drive
=======================

Create volume
-------------

Create volume in Varasto. For quota specify how much storage you want to allocate for
Varasto in Google Drive.

If you have unlimited storage in Google Drive
(as is for [G Suite Business](https://gsuite.google.com/pricing.html) which is cheap!),
just set an arbitrary limit.


Create folder in Drive
----------------------

Varasto stores all your blobs in Drive inside one folder (Drive has no limit of files per
folder and it is not a performance issue), so we need to create a folder.

If you named your volume "Mom" (I name my volumes by Futurama characters and Mom represents
an evil corporation.. maybe a coincidence?), I recommend you name your folder in Drive
"varasto-mom" for clarity (the name is technically arbitrary).

Open the folder to discover its ID from URL:

![](guide_setting-up-googledrive-gdrive-folder-id.png)

In my case my ID was `1znjU234YCcLW96u6_WrZtms2vFvGy55e`. You'll need this when mounting
the volume in Varasto.


Mount Drive as volume in Varasto
--------------------------------

Now mount the volume you created in Varasto, specifying `googledrive` as the blob driver type.

TODO: write document how to obtain the authentication details.

