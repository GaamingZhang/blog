# CMake的基本概念

## 概述

CMake是一个跨平台的开源构建系统生成工具,它使用平台无关的配置文件来生成特定平台的构建文件(如Makefile、Visual Studio项目文件等)。CMake已成为C/C++项目管理的事实标准。

**核心特点:**

- **跨平台**: 支持Linux、Windows、macOS等多种操作系统
- **编译器无关**: 支持GCC、Clang、MSVC等主流编译器
- **生成器模式**: 不直接构建项目,而是生成原生构建文件
- **易于维护**: 使用简洁的CMakeLists.txt文件描述项目结构
- **强大的依赖管理**: 自动处理库依赖和头文件路径

**CMake的工作流程:**

```
CMakeLists.txt → CMake → 构建文件(Makefile/VS项目) → 编译器 → 可执行文件
```

## CMake的安装与版本

### 安装CMake

**macOS:**

```bash
# 使用Homebrew安装
brew install cmake
```

### 查看版本

```bash
# 查看CMake版本
cmake --version

# 输出示例
# cmake version 3.28.1
```

**版本要求:**

现代C++项目建议使用CMake 3.15或更高版本,以获得更好的特性支持:

- CMake 3.12+: 支持`target_link_libraries`的改进语法
- CMake 3.15+: 更好的预编译头支持
- CMake 3.20+: 改进的C++20支持
- CMake 3.24+: 包管理器集成

## CMakeLists.txt基础

### 最简单的CMakeLists.txt

```cmake
# 指定CMake最低版本要求
cmake_minimum_required(VERSION 3.15)

# 定义项目名称和语言
project(HelloWorld CXX)

# 添加可执行文件
add_executable(hello main.cpp)
```

**对应的项目结构:**

```
project/
├── CMakeLists.txt
└── main.cpp
```

**main.cpp示例:**

```cpp
#include <iostream>

int main() {
    std::cout << "Hello, CMake!" << std::endl;
    return 0;
}
```

### 构建项目

```bash
# 创建构建目录(推荐做法,保持源码目录干净)
mkdir build
cd build

# 生成构建文件
cmake ..

# 编译项目
cmake --build .

# 或使用make(Linux/macOS)
make

# 运行程序
./hello
```

## CMake的核心概念

### 1. 项目(Project)

项目是CMake构建的顶层概念,使用`project()`命令定义。

```cmake
# 基本语法
project(项目名称 [语言...])

# 完整语法
project(MyProject
    VERSION 1.0.0
    DESCRIPTION "A sample project"
    LANGUAGES CXX C
)
```

**项目定义后的变量:**

```cmake
project(MyApp VERSION 1.2.3)

# 自动定义的变量
# ${PROJECT_NAME}           -> "MyApp"
# ${PROJECT_VERSION}        -> "1.2.3"
# ${PROJECT_VERSION_MAJOR}  -> "1"
# ${PROJECT_VERSION_MINOR}  -> "2"
# ${PROJECT_VERSION_PATCH}  -> "3"
# ${PROJECT_SOURCE_DIR}     -> 项目源码根目录
# ${PROJECT_BINARY_DIR}     -> 项目构建根目录
```

### 2. 目标(Target)

目标是CMake中最重要的概念,代表构建的产物,主要有三种类型:

**可执行文件目标:**

```cmake
# 创建可执行文件
add_executable(myapp main.cpp utils.cpp)
```

**库目标:**

```cmake
# 静态库
add_library(mylib STATIC lib.cpp)

# 动态库(共享库)
add_library(mylib SHARED lib.cpp)

# 接口库(仅头文件库)
add_library(mylib INTERFACE)
```

**自定义目标:**

```cmake
# 自定义命令目标
add_custom_target(docs
    COMMAND doxygen ${CMAKE_SOURCE_DIR}/Doxyfile
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
)
```

### 3. 变量(Variables)

CMake使用变量来存储配置信息和路径。

**定义和使用变量:**

```cmake
# 定义变量
set(MY_VAR "value")
set(SOURCE_FILES main.cpp utils.cpp helper.cpp)

# 使用变量
message("Variable value: ${MY_VAR}")
add_executable(myapp ${SOURCE_FILES})

# 列表操作
list(APPEND SOURCE_FILES extra.cpp)
list(REMOVE_ITEM SOURCE_FILES utils.cpp)
```

**常用的预定义变量:**

```cmake
# 目录相关
${CMAKE_SOURCE_DIR}         # 顶层CMakeLists.txt所在目录
${CMAKE_BINARY_DIR}         # 顶层构建目录
${CMAKE_CURRENT_SOURCE_DIR} # 当前CMakeLists.txt所在目录
${CMAKE_CURRENT_BINARY_DIR} # 当前构建目录

# 项目相关
${PROJECT_NAME}            # 项目名称
${PROJECT_SOURCE_DIR}      # 项目源码目录
${PROJECT_BINARY_DIR}      # 项目构建目录

# 系统相关
${CMAKE_SYSTEM_NAME}       # 操作系统名称(Linux/Windows/Darwin)
${CMAKE_SYSTEM_PROCESSOR}  # 处理器架构(x86_64/arm64)

# 编译器相关
${CMAKE_CXX_COMPILER}      # C++编译器路径
${CMAKE_C_COMPILER}        # C编译器路径
${CMAKE_CXX_COMPILER_ID}   # 编译器ID(GNU/Clang/MSVC)
```

### 4. 属性(Properties)

属性用于配置目标、目录或源文件的特性。

**目标属性:**

```cmake
# 设置C++标准
set_target_properties(myapp PROPERTIES
    CXX_STANDARD 17
    CXX_STANDARD_REQUIRED ON
)

# 设置输出名称
set_target_properties(mylib PROPERTIES
    OUTPUT_NAME "custom_name"
    VERSION 1.0.0
    SOVERSION 1
)

# 获取属性
get_target_property(OUT_NAME mylib OUTPUT_NAME)
message("Library output name: ${OUT_NAME}")
```

### 5. 生成器表达式(Generator Expressions)

生成器表达式在生成构建文件时求值,支持条件配置。

```cmake
# 基本语法: $<条件:真值>

# 根据构建类型添加不同的编译选项
target_compile_options(myapp PRIVATE
    $<$<CONFIG:Debug>:-g -O0>
    $<$<CONFIG:Release>:-O3>
)

# 根据编译器添加选项
target_compile_options(myapp PRIVATE
    $<$<CXX_COMPILER_ID:GNU>:-Wall -Wextra>
    $<$<CXX_COMPILER_ID:MSVC>:/W4>
)

# 平台相关的链接库
target_link_libraries(myapp PRIVATE
    $<$<PLATFORM_ID:Linux>:pthread>
    $<$<PLATFORM_ID:Windows>:ws2_32>
)
```

## 目标属性与依赖管理

### include目录管理

```cmake
# 为目标添加include目录
target_include_directories(mylib
    PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/include  # 公共接口头文件
    PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src      # 私有实现头文件
)

# PUBLIC: 该目标和使用该目标的其他目标都能访问
# PRIVATE: 仅该目标可以访问
# INTERFACE: 仅使用该目标的其他目标可以访问
```

**可见性说明:**

```cmake
# 示例项目结构
add_library(mylib lib.cpp)
target_include_directories(mylib
    PUBLIC include/         # 对外暴露的API头文件
    PRIVATE src/internal/   # 内部实现头文件
)

add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE mylib)
# myapp可以访问mylib的PUBLIC include目录
# myapp不能访问mylib的PRIVATE include目录
```

### 编译选项管理

```cmake
# 添加编译选项
target_compile_options(myapp PRIVATE
    -Wall           # 启用所有警告
    -Wextra         # 额外警告
    -Werror         # 警告视为错误
    -pedantic       # 严格标准检查
)

# 添加预处理器定义
target_compile_definitions(myapp PRIVATE
    DEBUG_MODE
    VERSION="1.0"
    $<$<CONFIG:Debug>:ENABLE_LOGGING>
)
```

### 链接库管理

```cmake
# 链接库
target_link_libraries(myapp PRIVATE mylib)

# 链接多个库
target_link_libraries(myapp
    PRIVATE
        mylib
        pthread
        m  # 数学库
)

# 链接系统库(FindXXX模块)
find_package(Threads REQUIRED)
target_link_libraries(myapp PRIVATE Threads::Threads)

find_package(Boost REQUIRED COMPONENTS filesystem)
target_link_libraries(myapp PRIVATE Boost::filesystem)
```

## 项目组织结构

### 单目录项目

```
simple_project/
├── CMakeLists.txt
├── main.cpp
└── utils.cpp
```

```cmake
cmake_minimum_required(VERSION 3.15)
project(SimpleProject CXX)

add_executable(myapp main.cpp utils.cpp)
```

### 多目录项目

```
complex_project/
├── CMakeLists.txt
├── src/
│   ├── CMakeLists.txt
│   └── main.cpp
├── lib/
│   ├── CMakeLists.txt
│   ├── include/
│   │   └── mylib.h
│   └── src/
│       └── mylib.cpp
└── tests/
    ├── CMakeLists.txt
    └── test_main.cpp
```

**顶层CMakeLists.txt:**

```cmake
cmake_minimum_required(VERSION 3.15)
project(ComplexProject VERSION 1.0.0 LANGUAGES CXX)

# 设置C++标准
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 添加子目录
add_subdirectory(lib)
add_subdirectory(src)
add_subdirectory(tests)
```

**lib/CMakeLists.txt:**

```cmake
# 创建库
add_library(mylib
    src/mylib.cpp
)

# 设置include目录
target_include_directories(mylib
    PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# 设置别名(推荐做法)
add_library(MyProject::mylib ALIAS mylib)
```

**src/CMakeLists.txt:**

```cmake
# 创建可执行文件
add_executable(myapp main.cpp)

# 链接库
target_link_libraries(myapp PRIVATE MyProject::mylib)
```

**tests/CMakeLists.txt:**

```cmake
# 启用测试
enable_testing()

# 创建测试可执行文件
add_executable(test_myapp test_main.cpp)
target_link_libraries(test_myapp PRIVATE MyProject::mylib)

# 添加测试
add_test(NAME MyTest COMMAND test_myapp)
```

### 现代CMake项目结构推荐

```
modern_project/
├── CMakeLists.txt
├── cmake/                    # CMake模块和脚本
│   └── FindSomeLib.cmake
├── include/                  # 公共头文件
│   └── myproject/
│       └── api.h
├── src/                      # 源文件
│   ├── CMakeLists.txt
│   ├── api.cpp
│   └── internal/
│       └── impl.cpp
├── tests/                    # 测试
│   ├── CMakeLists.txt
│   └── test_api.cpp
├── examples/                 # 示例
│   └── example.cpp
├── docs/                     # 文档
└── third_party/              # 第三方库
    └── some_lib/
```

## 常用CMake命令

### 文件操作

```cmake
# 搜索文件
file(GLOB SOURCES "src/*.cpp")
file(GLOB_RECURSE ALL_SOURCES "src/*.cpp")  # 递归搜索

# 注意: GLOB不推荐用于生产代码,因为添加新文件时CMake不会自动重新配置
# 推荐显式列出所有源文件

# 复制文件
file(COPY ${CMAKE_SOURCE_DIR}/config/ DESTINATION ${CMAKE_BINARY_DIR}/config)

# 配置文件(替换变量)
configure_file(config.h.in config.h)
```

### 条件判断

```cmake
# if语句
if(WIN32)
    message("Building on Windows")
elseif(UNIX)
    message("Building on Unix-like system")
    if(APPLE)
        message("Building on macOS")
    else()
        message("Building on Linux")
    endif()
endif()

# 变量判断
if(DEFINED MY_VAR)
    message("MY_VAR is defined: ${MY_VAR}")
endif()

if(MY_VAR STREQUAL "value")
    message("MY_VAR equals 'value'")
endif()

# 逻辑运算
if(VAR1 AND VAR2)
    message("Both variables are true")
endif()

if(NOT MY_VAR)
    message("MY_VAR is false or empty")
endif()
```

### 循环

```cmake
# foreach循环
set(ITEMS apple banana orange)
foreach(ITEM ${ITEMS})
    message("Item: ${ITEM}")
endforeach()

# 范围循环
foreach(i RANGE 5)
    message("Index: ${i}")  # 0, 1, 2, 3, 4, 5
endforeach()

# while循环
set(COUNT 0)
while(COUNT LESS 5)
    message("Count: ${COUNT}")
    math(EXPR COUNT "${COUNT} + 1")
endwhile()
```

### 函数和宏

```cmake
# 定义函数
function(my_function ARG1 ARG2)
    message("Function called with: ${ARG1}, ${ARG2}")
    set(RESULT "value" PARENT_SCOPE)  # 返回值到父作用域
endfunction()

# 调用函数
my_function("hello" "world")

# 定义宏(宏没有独立作用域)
macro(my_macro ARG)
    message("Macro called with: ${ARG}")
    set(RESULT "value")  # 直接在当前作用域设置
endmacro()

# 调用宏
my_macro("test")
```

### 消息输出

```cmake
# 不同级别的消息
message(STATUS "This is a status message")      # 普通信息
message(WARNING "This is a warning")            # 警告
message(SEND_ERROR "This is an error")          # 错误,继续执行
message(FATAL_ERROR "This is a fatal error")    # 致命错误,停止执行

# 调试信息
message(DEBUG "Debug information")
message(VERBOSE "Verbose information")
```

## 构建类型与配置

### 构建类型

CMake支持多种预定义的构建类型:

```cmake
# 设置默认构建类型
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING
        "Choose the type of build (Debug/Release/RelWithDebInfo/MinSizeRel)"
        FORCE
    )
endif()

# 构建类型说明:
# Debug: 包含调试信息,无优化 (-g -O0)
# Release: 优化,无调试信息 (-O3 -DNDEBUG)
# RelWithDebInfo: 优化+调试信息 (-O2 -g -DNDEBUG)
# MinSizeRel: 最小化大小 (-Os -DNDEBUG)
```

**使用方法:**

```bash
# 指定构建类型
cmake -DCMAKE_BUILD_TYPE=Debug ..
cmake -DCMAKE_BUILD_TYPE=Release ..

# 多配置生成器(Visual Studio, Xcode)
cmake --build . --config Debug
cmake --build . --config Release
```

### 编译选项配置

```cmake
# 设置C++标准
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)  # 禁用编译器扩展

# 全局编译选项
add_compile_options(-Wall -Wextra)

# 根据编译器添加选项
if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    add_compile_options(-fPIC)
elseif(MSVC)
    add_compile_options(/W4)
endif()

# 根据构建类型添加选项
set(CMAKE_CXX_FLAGS_DEBUG "${CMAKE_CXX_FLAGS_DEBUG} -fsanitize=address")
set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -march=native")
```

### 选项定义

```cmake
# 定义用户可配置的选项
option(BUILD_SHARED_LIBS "Build shared libraries" ON)
option(ENABLE_TESTING "Enable testing" ON)
option(USE_OPENMP "Enable OpenMP support" OFF)

# 使用选项
if(BUILD_SHARED_LIBS)
    add_library(mylib SHARED lib.cpp)
else()
    add_library(mylib STATIC lib.cpp)
endif()

if(ENABLE_TESTING)
    enable_testing()
    add_subdirectory(tests)
endif()
```

**命令行设置选项:**

```bash
cmake -DBUILD_SHARED_LIBS=OFF -DENABLE_TESTING=ON ..
```

## 依赖管理

### find_package

`find_package`是CMake查找和使用外部库的标准方式。

```cmake
# 查找包(必需)
find_package(Threads REQUIRED)

# 查找包(可选)
find_package(OpenMP)

# 查找特定版本
find_package(Boost 1.70 REQUIRED)

# 查找包的特定组件
find_package(Boost REQUIRED COMPONENTS filesystem system)

# 使用找到的包
if(Boost_FOUND)
    target_include_directories(myapp PRIVATE ${Boost_INCLUDE_DIRS})
    target_link_libraries(myapp PRIVATE ${Boost_LIBRARIES})
endif()

# 现代CMake方式(推荐)
find_package(Boost REQUIRED COMPONENTS filesystem)
target_link_libraries(myapp PRIVATE Boost::filesystem)
```

### FetchContent (CMake 3.11+)

在配置时下载和集成外部项目。

```cmake
include(FetchContent)

# 声明外部内容
FetchContent_Declare(
    googletest
    GIT_REPOSITORY https://github.com/google/googletest.git
    GIT_TAG v1.14.0
)

# 使用外部内容
FetchContent_MakeAvailable(googletest)

# 现在可以使用googletest
add_executable(test_myapp test.cpp)
target_link_libraries(test_myapp PRIVATE gtest gtest_main)
```

### ExternalProject

在构建时下载和构建外部项目。

```cmake
include(ExternalProject)

ExternalProject_Add(
    external_lib
    GIT_REPOSITORY https://github.com/user/lib.git
    GIT_TAG v1.0
    PREFIX ${CMAKE_BINARY_DIR}/external
    CMAKE_ARGS
        -DCMAKE_INSTALL_PREFIX=<INSTALL_DIR>
        -DCMAKE_BUILD_TYPE=${CMAKE_BUILD_TYPE}
    BUILD_COMMAND cmake --build . --config ${CMAKE_BUILD_TYPE}
    INSTALL_COMMAND cmake --install .
)
```

### 子模块方式

```cmake
# 添加子模块目录
add_subdirectory(third_party/some_lib)

# 使用子模块的目标
target_link_libraries(myapp PRIVATE some_lib)
```

## 安装与打包

### 安装配置

```cmake
# 安装可执行文件
install(TARGETS myapp
    RUNTIME DESTINATION bin
)

# 安装库
install(TARGETS mylib
    LIBRARY DESTINATION lib        # 动态库
    ARCHIVE DESTINATION lib        # 静态库
    RUNTIME DESTINATION bin        # DLL(Windows)
    INCLUDES DESTINATION include   # 头文件搜索路径
)

# 安装头文件
install(DIRECTORY include/
    DESTINATION include
    FILES_MATCHING PATTERN "*.h"
)

# 安装文件
install(FILES README.md LICENSE
    DESTINATION share/doc/myproject
)

# 安装配置文件
install(FILES config/myapp.conf
    DESTINATION etc/myapp
)
```

**使用方法:**

```bash
# 构建
cmake --build .

# 安装到默认位置(/usr/local)
sudo cmake --install .

# 安装到指定位置
cmake --install . --prefix /opt/myapp

# 或在配置时指定
cmake -DCMAKE_INSTALL_PREFIX=/opt/myapp ..
```

### CPack打包

```cmake
# 设置打包信息
set(CPACK_PACKAGE_NAME "MyProject")
set(CPACK_PACKAGE_VERSION "1.0.0")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "My awesome project")
set(CPACK_PACKAGE_VENDOR "My Company")

# 设置打包格式
set(CPACK_GENERATOR "TGZ;ZIP")  # Linux: TGZ, Windows: ZIP

# 设置DEB包信息(Debian/Ubuntu)
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "Your Name <email@example.com>")
set(CPACK_DEBIAN_PACKAGE_DEPENDS "libboost-all-dev")

# 设置RPM包信息(RedHat/CentOS)
set(CPACK_RPM_PACKAGE_LICENSE "MIT")
set(CPACK_RPM_PACKAGE_REQUIRES "boost-devel")

# 包含CPack
include(CPack)
```

**生成安装包:**

```bash
# 构建项目
cmake --build .

# 生成安装包
cpack

# 生成特定格式的包
cpack -G ZIP
cpack -G DEB
cpack -G RPM
```

## 实用示例

### 完整的库项目模板

```cmake
cmake_minimum_required(VERSION 3.15)

# 项目信息
project(MyLibrary
    VERSION 1.0.0
    DESCRIPTION "A useful library"
    LANGUAGES CXX
)

# 设置C++标准
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# 选项
option(BUILD_SHARED_LIBS "Build shared library" ON)
option(BUILD_TESTS "Build tests" ON)
option(BUILD_EXAMPLES "Build examples" ON)

# 包含模块
include(GNUInstallDirs)

# 创建库
add_library(mylib
    src/mylib.cpp
    src/utils.cpp
)

# 设置别名
add_library(MyLib::mylib ALIAS mylib)

# 设置属性
set_target_properties(mylib PROPERTIES
    VERSION ${PROJECT_VERSION}
    SOVERSION 1
    PUBLIC_HEADER include/mylib/mylib.h
)

# 包含目录
target_include_directories(mylib
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>
    PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# 编译选项
target_compile_options(mylib PRIVATE
    $<$<CXX_COMPILER_ID:GNU,Clang>:-Wall -Wextra -Wpedantic>
    $<$<CXX_COMPILER_ID:MSVC>:/W4>
)

# 链接库
find_package(Threads REQUIRED)
target_link_libraries(mylib
    PUBLIC
        Threads::Threads
    PRIVATE
        m  # 数学库(仅Linux)
)

# 安装
install(TARGETS mylib
    EXPORT MyLibTargets
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
    PUBLIC_HEADER DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}/mylib
)

# 导出目标
install(EXPORT MyLibTargets
    FILE MyLibTargets.cmake
    NAMESPACE MyLib::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/MyLib
)

# 生成Config文件
include(CMakePackageConfigHelpers)
configure_package_config_file(
    ${CMAKE_CURRENT_SOURCE_DIR}/cmake/MyLibConfig.cmake.in
    ${CMAKE_CURRENT_BINARY_DIR}/MyLibConfig.cmake
    INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/MyLib
)

# 生成Version文件
write_basic_package_version_file(
    ${CMAKE_CURRENT_BINARY_DIR}/MyLibConfigVersion.cmake
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY SameMajorVersion
)

# 安装Config文件
install(FILES
    ${CMAKE_CURRENT_BINARY_DIR}/MyLibConfig.cmake
    ${CMAKE_CURRENT_BINARY_DIR}/MyLibConfigVersion.cmake
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/MyLib
)

# 测试
if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# 示例
if(BUILD_EXAMPLES)
    add_subdirectory(examples)
endif()
```

### 跨平台配置示例

```cmake
# 平台检测
if(WIN32)
    # Windows特定配置
    target_compile_definitions(myapp PRIVATE PLATFORM_WINDOWS)
    target_sources(myapp PRIVATE src/windows/platform.cpp)
    target_link_libraries(myapp PRIVATE ws2_32)
    
elseif(APPLE)
    # macOS特定配置
    target_compile_definitions(myapp PRIVATE PLATFORM_MACOS)
    target_sources(myapp PRIVATE src/macos/platform.mm)
    find_library(COCOA_LIBRARY Cocoa)
    target_link_libraries(myapp PRIVATE ${COCOA_LIBRARY})
    
elseif(UNIX)
    # Linux特定配置
    target_compile_definitions(myapp PRIVATE PLATFORM_LINUX)
    target_sources(myapp PRIVATE src/linux/platform.cpp)
    target_link_libraries(myapp PRIVATE pthread dl)
endif()

# 架构检测
if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|AMD64")
    target_compile_definitions(myapp PRIVATE ARCH_X64)
elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|ARM64")
    target_compile_definitions(myapp PRIVATE ARCH_ARM64)
endif()

# 编译器检测
if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
    target_compile_options(myapp PRIVATE -fno-rtti)
elseif(CMAKE_CXX_COMPILER_ID STREQUAL "Clang")
    target_compile_options(myapp PRIVATE -fno-rtti)
elseif(CMAKE_CXX_COMPILER_ID STREQUAL "MSVC")
    target_compile_options(myapp PRIVATE /GR-)
endif()
```

## 常见问题

### 1. 什么是Out-of-Source构建?为什么推荐使用?

**Out-of-Source构建**是指将构建产物(中间文件、可执行文件等)生成到源代码目录之外的独立目录中,而不是在源码目录中直接构建。

**In-Source构建(不推荐):**

```bash
cd project
cmake .        # 在源码目录构建
make
```

这会在源码目录中生成大量构建文件,污染源码目录。

**Out-of-Source构建(推荐):**

```bash
cd project
mkdir build    # 创建独立的构建目录
cd build
cmake ..       # 在build目录中构建
make
```

**优势:**

1. **保持源码目录干净**: 所有构建产物都在build目录中,不影响源码
2. **支持多种构建配置**: 可以创建多个构建目录用于不同配置

```bash
mkdir build-debug
mkdir build-release
cd build-debug && cmake -DCMAKE_BUILD_TYPE=Debug ..
cd ../build-release && cmake -DCMAKE_BUILD_TYPE=Release ..
```

3. **方便清理**: 删除build目录即可完全清理,不用担心遗留文件
4. **避免版本冲突**: 构建文件不会被提交到版本控制系统

**最佳实践:**

在`.gitignore`中添加:

```
build/
build-*/
cmake-build-*/
```

### 2. target_link_libraries中的PUBLIC、PRIVATE和INTERFACE有什么区别?

这三个关键字控制依赖关系的传播范围,是现代CMake的核心概念。

**定义:**

- **PRIVATE**: 依赖仅用于目标本身的实现,不传播给链接该目标的其他目标
- **PUBLIC**: 依赖既用于目标本身,也传播给链接该目标的其他目标
- **INTERFACE**: 依赖不用于目标本身,仅传播给链接该目标的其他目标

**实例说明:**

```cmake
# 假设有三个库: A, B, C

# 库A的实现
add_library(A a.cpp)
target_link_libraries(A PRIVATE B)     # A内部使用B,但不暴露B的接口
target_link_libraries(A PUBLIC C)      # A使用C,并且A的头文件也暴露了C的类型

# 应用程序链接A
add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE A)

# 结果:
# - myapp会链接A和C(因为C是A的PUBLIC依赖)
# - myapp不会链接B(因为B是A的PRIVATE依赖)
```

**具体场景:**

```cmake
# 场景1: 库的头文件中使用了依赖(使用PUBLIC)
# mylib.h
#include <boost/filesystem.hpp>  // 头文件暴露了Boost类型
class MyLib {
    boost::filesystem::path getPath();
};

# CMakeLists.txt
target_link_libraries(mylib PUBLIC Boost::filesystem)
# 使用mylib的代码也需要访问Boost::filesystem


# 场景2: 库仅在实现中使用依赖(使用PRIVATE)
# mylib.cpp
#include <zlib.h>  // 仅在实现中压缩数据
void MyLib::compress() {
    // 使用zlib...
}

# CMakeLists.txt
target_link_libraries(mylib PRIVATE ZLIB::ZLIB)
# 使用mylib的代码不需要知道zlib的存在


# 场景3: 仅头文件库(使用INTERFACE)
add_library(header_only_lib INTERFACE)
target_include_directories(header_only_lib INTERFACE include/)
# 该库本身没有源文件,仅提供头文件给其他目标使用
```

**选择原则:**

- 实现细节使用PRIVATE
- 公共API涉及的依赖使用PUBLIC
- 仅头文件库或转发依赖使用INTERFACE

### 3. find_package是如何查找库的?如何编写自定义的Find模块?

**find_package的查找机制:**

CMake通过以下方式查找包:

**1. Config模式(优先):**

查找`<PackageName>Config.cmake`或`<package-name>-config.cmake`文件。

搜索路径:
```
<PackageName>_DIR
CMAKE_PREFIX_PATH/lib/cmake/<PackageName>
CMAKE_INSTALL_PREFIX/lib/cmake/<PackageName>
/usr/local/lib/cmake/<PackageName>
/usr/lib/cmake/<PackageName>
```

**2. Module模式(备用):**

查找`Find<PackageName>.cmake`模块文件。

搜索路径:
```
CMAKE_MODULE_PATH
CMAKE安装目录/share/cmake/Modules/
```

**使用示例:**

```cmake
# 指定查找路径
list(APPEND CMAKE_PREFIX_PATH "/opt/mylib")
list(APPEND CMAKE_MODULE_PATH "${CMAKE_SOURCE_DIR}/cmake")

# 查找包
find_package(MyLib REQUIRED)

# 使用找到的包
target_link_libraries(myapp PRIVATE MyLib::MyLib)
```

**编写自定义Find模块:**

创建`cmake/FindMyLib.cmake`:

```cmake
# FindMyLib.cmake

# 查找头文件
find_path(MyLib_INCLUDE_DIR
    NAMES mylib.h
    PATHS
        /usr/include
        /usr/local/include
        ${MyLib_ROOT}/include
)

# 查找库文件
find_library(MyLib_LIBRARY
    NAMES mylib
    PATHS
        /usr/lib
        /usr/local/lib
        ${MyLib_ROOT}/lib
)

# 设置版本信息(可选)
if(MyLib_INCLUDE_DIR)
    file(READ "${MyLib_INCLUDE_DIR}/mylib.h" VERSION_HEADER)
    string(REGEX MATCH "MYLIB_VERSION \"([0-9.]+)\"" _ ${VERSION_HEADER})
    set(MyLib_VERSION ${CMAKE_MATCH_1})
endif()

# 使用FindPackageHandleStandardArgs
include(FindPackageHandleStandardArgs)
find_package_handle_standard_args(MyLib
    REQUIRED_VARS
        MyLib_LIBRARY
        MyLib_INCLUDE_DIR
    VERSION_VAR MyLib_VERSION
)

# 创建导入目标(现代CMake推荐)
if(MyLib_FOUND AND NOT TARGET MyLib::MyLib)
    add_library(MyLib::MyLib UNKNOWN IMPORTED)
    set_target_properties(MyLib::MyLib PROPERTIES
        IMPORTED_LOCATION "${MyLib_LIBRARY}"
        INTERFACE_INCLUDE_DIRECTORIES "${MyLib_INCLUDE_DIR}"
    )
endif()

# 标记为高级变量(GUI中隐藏)
mark_as_advanced(MyLib_INCLUDE_DIR MyLib_LIBRARY)
```

**使用自定义Find模块:**

```cmake
# CMakeLists.txt
list(APPEND CMAKE_MODULE_PATH "${CMAKE_SOURCE_DIR}/cmake")

find_package(MyLib REQUIRED)

target_link_libraries(myapp PRIVATE MyLib::MyLib)
```

### 4. 如何在CMake中正确管理C++标准(C++11/14/17/20)?

**现代CMake方法(推荐):**

```cmake
# 方法1: 全局设置(影响所有目标)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)   # 强制要求,不满足则报错
set(CMAKE_CXX_EXTENSIONS OFF)         # 禁用编译器扩展(如GNU扩展)

# 方法2: 针对特定目标设置
add_executable(myapp main.cpp)
set_target_properties(myapp PROPERTIES
    CXX_STANDARD 17
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF
)

# 方法3: 使用target_compile_features(最灵活)
add_executable(myapp main.cpp)
target_compile_features(myapp PRIVATE
    cxx_std_17           # 要求C++17
    # 或者指定具体特性
    cxx_lambda           # Lambda表达式
    cxx_auto_type        # auto关键字
    cxx_range_for        # 范围for循环
)
```

**不同标准的设置:**

```cmake
# C++11
set(CMAKE_CXX_STANDARD 11)

# C++14
set(CMAKE_CXX_STANDARD 14)

# C++17
set(CMAKE_CXX_STANDARD 17)

# C++20
set(CMAKE_CXX_STANDARD 20)

# C++23 (CMake 3.20+)
set(CMAKE_CXX_STANDARD 23)
```

**跨平台兼容性检查:**

```cmake
# 检查编译器是否支持C++17
include(CheckCXXCompilerFlag)
check_cxx_compiler_flag("-std=c++17" COMPILER_SUPPORTS_CXX17)

if(NOT COMPILER_SUPPORTS_CXX17)
    message(FATAL_ERROR "Compiler doesn't support C++17")
endif()

# 或使用target_compile_features自动检查
add_executable(myapp main.cpp)
target_compile_features(myapp PRIVATE cxx_std_17)
# 如果编译器不支持,会自动报错
```

**避免的旧方法:**

```cmake
# 不推荐:手动添加编译标志
add_compile_options(-std=c++17)  # 不跨平台

# 不推荐:使用CMAKE_CXX_FLAGS
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++17")
```

**库提供者的最佳实践:**

```cmake
# 库应该指定最低要求,不强制特定版本
add_library(mylib lib.cpp)

# 要求至少C++11
target_compile_features(mylib PUBLIC cxx_std_11)

# 使用该库的项目可以选择更高标准
add_executable(myapp main.cpp)
target_compile_features(myapp PRIVATE cxx_std_17)
target_link_libraries(myapp PRIVATE mylib)
# myapp使用C++17,mylib使用C++11,完全兼容
```

### 5. 如何使用CMake构建和集成第三方库?有哪些常用方法?

CMake提供了多种集成第三方库的方法,各有优缺点:

**方法1: find_package(系统安装的库)**

适用于已经安装在系统中的库。

```cmake
# 查找系统中的Boost库
find_package(Boost 1.70 REQUIRED COMPONENTS filesystem system)

target_link_libraries(myapp PRIVATE
    Boost::filesystem
    Boost::system
)

# 查找OpenSSL
find_package(OpenSSL REQUIRED)
target_link_libraries(myapp PRIVATE OpenSSL::SSL OpenSSL::Crypto)
```

**优点:** 快速、不增加构建时间
**缺点:** 需要预先安装、版本管理困难、跨平台部署复杂

**方法2: FetchContent(源码集成,配置时下载)**

适用于需要从源码构建的小型库。

```cmake
include(FetchContent)

# 集成fmt库
FetchContent_Declare(
    fmt
    GIT_REPOSITORY https://github.com/fmtlib/fmt.git
    GIT_TAG 9.1.0
)
FetchContent_MakeAvailable(fmt)

target_link_libraries(myapp PRIVATE fmt::fmt)

# 集成多个库
FetchContent_Declare(
    spdlog
    GIT_REPOSITORY https://github.com/gabime/spdlog.git
    GIT_TAG v1.12.0
)

FetchContent_Declare(
    json
    GIT_REPOSITORY https://github.com/nlohmann/json.git
    GIT_TAG v3.11.2
)

FetchContent_MakeAvailable(spdlog json)

target_link_libraries(myapp PRIVATE
    spdlog::spdlog
    nlohmann_json::nlohmann_json
)
```

**优点:** 版本精确控制、自包含、易于复现
**缺点:** 增加配置时间、每个构建目录独立下载

**方法3: add_subdirectory(Git子模块)**

将第三方库作为子模块添加到项目中。

```bash
# 添加Git子模块
git submodule add https://github.com/google/googletest.git third_party/googletest
git submodule update --init --recursive
```

```cmake
# CMakeLists.txt
add_subdirectory(third_party/googletest)

add_executable(test_myapp test.cpp)
target_link_libraries(test_myapp PRIVATE gtest gtest_main)
```

**优点:** 离线可用、构建快速、版本锁定
**缺点:** Git子模块管理复杂、占用仓库空间

**方法4: ExternalProject(构建时下载)**

在构建阶段下载和编译外部项目。

```cmake
include(ExternalProject)

ExternalProject_Add(
    libzip_external
    GIT_REPOSITORY https://github.com/nih-at/libzip.git
    GIT_TAG v1.9.2
    PREFIX ${CMAKE_BINARY_DIR}/external
    CMAKE_ARGS
        -DCMAKE_INSTALL_PREFIX=<INSTALL_DIR>
        -DCMAKE_BUILD_TYPE=${CMAKE_BUILD_TYPE}
        -DBUILD_SHARED_LIBS=OFF
    BUILD_COMMAND cmake --build . --config ${CMAKE_BUILD_TYPE}
    INSTALL_COMMAND cmake --install .
)

# 使用安装的库
ExternalProject_Get_Property(libzip_external INSTALL_DIR)
add_library(libzip STATIC IMPORTED)
set_target_properties(libzip PROPERTIES
    IMPORTED_LOCATION ${INSTALL_DIR}/lib/libzip.a
    INTERFACE_INCLUDE_DIRECTORIES ${INSTALL_DIR}/include
)
add_dependencies(libzip libzip_external)

target_link_libraries(myapp PRIVATE libzip)
```

**优点:** 不影响配置时间、适合大型库
**缺点:** 配置复杂、难以调试

**方法5: 包管理器(Conan/vcpkg)**

使用专业的C++包管理器。

**使用vcpkg:**

```bash
# 安装vcpkg
git clone https://github.com/microsoft/vcpkg.git
./vcpkg/bootstrap-vcpkg.sh

# 安装库
./vcpkg/vcpkg install fmt boost-filesystem
```

```cmake
# CMakeLists.txt
set(CMAKE_TOOLCHAIN_FILE "${VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake")

find_package(fmt CONFIG REQUIRED)
find_package(Boost REQUIRED COMPONENTS filesystem)

target_link_libraries(myapp PRIVATE fmt::fmt Boost::filesystem)
```

**使用Conan:**

创建`conanfile.txt`:

```ini
[requires]
fmt/9.1.0
boost/1.81.0

[generators]
CMakeDeps
CMakeToolchain
```

```bash
# 安装依赖
conan install . --output-folder=build --build=missing

# 配置项目
cmake -S . -B build -DCMAKE_TOOLCHAIN_FILE=build/conan_toolchain.cmake
cmake --build build
```

**选择建议:**

- **小型纯头文件库**: FetchContent
- **中型库需要源码构建**: FetchContent或add_subdirectory
- **大型库或多项目共享**: 系统安装+find_package或vcpkg/Conan
- **需要离线构建**: add_subdirectory(Git子模块)
- **复杂依赖关系**: vcpkg或Conan包管理器