#!/usr/bin/perl

use strict;
use warnings;
use File::Copy;

my $file = $ARGV[0] or die "Usage: $0 <file_to_modify>\n";

local @ARGV = ($file);
local $^I = '.bak';
while (<>) {
    $_ =~ tr/ /-/;
    $_ = lc($_);
    print;
}

unlink "$file.bak";

print "File has been modified.\n";
