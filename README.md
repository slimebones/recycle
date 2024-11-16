# recycle
Cross-platform command-line solution to file recycling (aka trash).

## Usage
* `recycle store PATH` - to trash files at PATH
* `recycle list [PATH]` - to list trashed files at PATH directory or it's subdirectories. Defaults to current working directory. Parent jumping with dots, Parent jumping with dots, like `../somediratparent` is not supported, maybe only for now.
* `recycle recover ID` - recover file by it's id, in the current working directory. To find out trashed files ids, call `recycle list` being in the same directory, where recover is desired.

## Advanced notes
* everything is operated under `~/.recycle/`
