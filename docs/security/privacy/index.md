Who can access my files?
------------------------

Only you (and the people you explicitly choose to share with) have access to your files.


Does Varasto phone home?
------------------------

No, except for update checking.


### Update checking

This is done to tell you if there is a better version available (displayed in server UI),
and possibly to alert you if there are critical security updates available.

This version check request doesn't contain any additional data other than what is required
to do the check:

| Data | Example | Used for analytics[^1] | Why we send this |
|------|---------|--------------------|------------------|
| OS | Linux | ☑️ | So we can tell you the latest version for your OS |
| Architecture | x86-64 | ☑️ | So we can tell you the latest version for your architecture |
| Current version | 20200418_1637_fa31fb5e | ☐ | So we can tell you if your version contains critical vulnerabilities |
| IP address | 84.15.186.115 | ☐ | One can't check for updates (or use the internet) without revealing one's IP address |

!!! tip
	You can audit the
	[version checking code](https://github.com/function61/varasto/blob/6eb3f4d6f18ce61be453291ab644fe8ef64aad62/pkg/stoserver/updatechecker.go#L63)
	yourself.


The data we record about Varasto users
--------------------------------------

### Self-hosted Varasto

Nothing.


### Cloud-hosted Varasto

Since this is a paid offering, we have to keep your billing details on file, and of course
an email to reach out to you for important updates.


### Varasto website visitors

We don't use any analytics except for aggregate request counts.


### Varasto newsletter

If you sign up for [Varasto newsletter](https://buttondown.email/varasto), we can see your
email address. We won't sell your email address ever, and only use it for purposes stated
in the newsletter's description.


[^1]: To gather metrics like "Active Varasto users on Linux/x86-64" so we know which
      platforms to focus our development efforts on.
