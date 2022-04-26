import argparse


def get_args():
    parser = argparse.ArgumentParser(description='')
    parser.add_argument('vulkan_xml', type=str,
                        help='The vulkan XML to parse')
    parser.add_argument('output_location', type=str,
                        help='The location to output the files')
    args = parser.parse_args()
    return args
