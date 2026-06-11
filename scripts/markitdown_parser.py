import sys
import argparse
from markitdown import MarkItDown

def parse_file(file_path):
    try:
        md = MarkItDown()
        result = md.convert(file_path)
        print(result.text_content)
    except Exception as e:
        print(f"Error parsing file: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Parse file to Markdown using MarkItDown")
    parser.add_argument("file_path", help="Path to the file to parse")
    args = parser.parse_args()
    
    parse_file(args.file_path)
