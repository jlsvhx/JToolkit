import os
import subprocess

directory = r'E:\experiment\libavif-v1.0.4-avifenc-avifdec-windows'

def check_color_depth(directory):
    for root, _, files in os.walk(directory):
        for file in files:
            if file.lower().endswith(('jpg', 'jpeg', 'png', 'tiff', 'bmp', 'gif')):
                file_path = os.path.join(root, file)
                result = subprocess.run(['exiftool', '-BitsPerSample', '-BitDepth', file_path], capture_output=True,
                                        text=True)
                bits_per_sample = None
                bit_depth = None

                for line in result.stdout.splitlines():
                    if 'Bits Per Sample' in line:
                        bits_per_sample = int(line.split(':')[-1].strip())
                    elif 'Bit Depth' in line:
                        bit_depth = int(line.split(':')[-1].strip())

                if bits_per_sample == 10 or bit_depth == 10:
                    print(f'{file_path} has a color depth of 10 bits.')
                else:
                    continue


# Example usage
check_color_depth(directory)