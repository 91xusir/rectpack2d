import os
from PIL import Image, ImageChops

def is_image_different(img1_path, img2_path):
    try:
        img1 = Image.open(img1_path).convert("RGBA")
        img2 = Image.open(img2_path).convert("RGBA")

        if img1.size != img2.size:
            return True  # 尺寸不同即不同

        diff = ImageChops.difference(img1, img2)
        return diff.getbbox() is not None  # 如果有差异区域，返回True
    except Exception as e:
        print(f"对比失败: {img1_path} <-> {img2_path}, 错误: {e}")
        return True

def compare_folders(input_dir, output_dir):
    different_files = []
    for filename in os.listdir(input_dir):
        input_path = os.path.join(input_dir, filename)
        output_path = os.path.join(output_dir, filename)

        if not os.path.isfile(output_path):
            print(f"缺失文件: {output_path}")
            continue

        if is_image_different(input_path, output_path):
            different_files.append(filename)

    return different_files

# 示例用法
if __name__ == "__main__":
    input_folder = "input"     # 替换为你的实际路径
    output_folder = "output"   # 替换为你的实际路径

    diffs = compare_folders(input_folder, output_folder)
    if diffs:
        print("以下图片不一致：")
        for name in diffs:
            print(" -", name)
    else:
        print("所有图片一致 ✅")
