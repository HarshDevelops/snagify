import sys
from .installer import run

def main():
    raise SystemExit(run(sys.argv[1:]))

if __name__ == "__main__":
    main()
