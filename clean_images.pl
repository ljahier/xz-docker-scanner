#!/usr/bin/perl

use strict;
use warnings;
use YAML::XS 'LoadFile';
use YAML::XS 'Dump';

# Make sure an input file is provided
my $input_file = shift or die "Usage: $0 <input_file.yaml>\n";
my $output_file = "modified_$input_file";

# Load the YAML file
my $data = LoadFile($input_file);

# Recursively process the data structure
sub process {
    my ($element) = @_;
    if (ref $element eq 'HASH') {
        foreach my $key (keys %$element) {
            process($element->{$key});
        }
    } elsif (ref $element eq 'ARRAY') {
        foreach my $item (@$element) {
            process($item);
        }
    } elsif (!ref $element) {
        $element =~ tr/ /-/;      # Replace spaces with dashes
        $element = lc($element);  # Convert to lowercase
        return $element;
    }
}

# Process and modify the loaded data
process($data);

# Save the modified data to a new file
open my $fh, '>', $output_file or die "Could not open file '$output_file': $!";
print $fh Dump($data);
close $fh;

print "Modified YAML has been written to '$output_file'\n";
