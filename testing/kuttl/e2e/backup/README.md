## Backing up a cluster with the CLI

(1) 00-01
* Create a cluster with no manual backup in the spec
* Check the command on the backup pod created after startup -- it should match what PGO gives

(2) 02-03
* Trigger a backup through the CLI with a flag set
* Check the command on the backup pod -- it should have the flag set

(3) 04-05
* Trigger a backup through the CLI with a flag with longer options
* Check the command on the backup pod -- it should have the flag set with longer options

(4) 06-07
* Trigger a backup through the CLI with multiple flags
* Check the command on the backup pod -- it should have multiple options

(5) 08-09
* Trigger a backup through the CLI with no flags
* Check the command on the backup pod -- it should have multiple options (the last set)

(6) 10-12
* Update the spec through KUTTL, changing the ownership of that field
* Trigger a backup through the CLI with different flags
* Check the command on the backup pod -- it should have the different flags
TODO(benjaminjb): is this truly the behavior we want?
