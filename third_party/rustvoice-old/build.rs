fn main() {
    // 设置 ONNX Runtime 库的路径
    println!("cargo:rustc-link-search=native=D:/onnxruntime/lib");

    // 链接 ONNX Runtime 库
    println!("cargo:rustc-link-lib=static=onnxruntime");
}
