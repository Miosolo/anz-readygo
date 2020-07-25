# Golang 学习笔记

## 前言
这篇笔记是我在完成项目全部代码后所写，主要着眼于我在项目中需要注意的、以及切实提升了项目质量和开发效率的 Golang 特性，所以也许不够全面，但我自认为总结性和应用性较强，这也正是笔记之所应承载的作用。
在接下来的内容中，我将在**语言特性（相较于C语言）、数据结构、并发控制、错误处理**等方面，对我所学习的 Golang 内容进行总结，并对我在项目中的实际应用进行说明。

## 语言特性
由于 Golang 是由 Google 内部孵化，最早用于网络服务开发；而此类工程师在 Google 大多具有 C 和 C++ 基础，所以 Golang 的语法结构上与 C 较为相似，更有“21世纪的 C 语言”之称。加之我有一些 C/C++ 背景，所以我将以一个 C/C++ 向 Golang 过渡的角度，主要讲 Go 相较于它们的特性。

### 变量声明与赋值
在 Golang 中，进行变量声明可以有以下的几种方式：
- 一次声明一个变量：`var identifier type`；
- 一次声明多个变量：`var identifier1, identifier2 type`；
- 使用初始化声明，由编译器自动推测类型并赋值：`var := ...`；
- 使用变量列表将多钟类型的变量赋值合并（在定义全局变量时格外有用），如下。
```go
  var(
    id1 id2 type1
    id3 type2
    id4 id5 type3
  )
```

在进行变量操作时，需要注意的点为：
- 变量不能重复声明，在使用 `:=` 进行声明时需要注意同作用域的同名变量不能再次使用 `:=` ；
- 将多返回值赋值给多个变量时，若存在新变量，依然可以使用 `:=` ；
- 在分支语句声明的变量只在自身作用域有效，即使每个分支都进行了同名声明；
- Golang 不推荐 `var identifier type = ...` 的写法，应当尽量使用编译器推断类型。

Golang 推荐使用驼峰式命名法，并默认首字母大写的变量为对包外可见(exported)变量，小写的变量名对于外部不可见。**需要注意的是，对于 exported struct，其中的每个成员若对外包可见，也必须首字母大写**。对于一些以无类型指针为参数，对其内容进行修改的函数（如数据库驱动程序的数据写入函数）来说，调用它们需要注意变量可见性，否则会写入失败返回全0。

### 分支语法
#### `if` 语句
`if` 语句由布尔表达式后紧跟一个或多个语句组成，Golang 中if语句的语法如下：
```go
if 布尔表达式 {
   /* 在布尔表达式为 true 时执行 */
} [else ...]
```
Golang 为 `if` 语句也添加了以下的功能和约束：
- 大括号{}必须存在，即使只有一行语句；
- 在if之后，条件语句之前，可以添加变量初始化语句，使用 `;` 进行分隔；
- 在有返回值的函数中，最终的return不能在条件语句中。

#### `switch` 语句
switch 语句用于基于不同条件执行不同动作，每一个 case 分支都是唯一的，从上至下逐一测试，直到匹配为止。**`switch` 默认情况下 case 最后自带 `break` 语句，匹配成功后就不会执行其他 case，如果我们需要执行后面的 case，可以使用 `fallthrough`** ，其语法如下：
```go
switch var1 {
    case val1:
        ...
    case val2, val3:
        ...
    default:
        ...
}
```
`switch` 语句还可以被用于 type-switch 来判断某个 interface 变量中实际存储的变量类型。Type Switch 语法格式如下：
```go
switch x.(type){
    case type:
       statement(s);      
    case type:
       statement(s); 
    /* 你可以定义任意个数的case */
    default: /* 可选 */
       statement(s);
}
```

#### `select` 语句
`select` 是 Golang 独有的一个控制结构，类似于用于通信的 `switch` 语句。每个 case 必须是一个通信操作，要么是发送要么是接收。`select` 随机执行一个可运行的 case。如果没有 case 可运行且没有 `default` case，它将阻塞，直到有 case 可运行。其语法如下：
```go
select {
    case communication clause  :
       statement(s);      
    case communication clause  :
       statement(s); 
    /* 你可以定义任意数量的 case */
    default : /* 可选 */
       statement(s);
}
```
而从原理上来讲，`select` 会循环检测条件，如果有 case 满足则执行并退出，否则一直循环检测。

### 函数
在我看来，**Golang 中的函数结构与 C 语言较为类似，但在此基础上添加了方法、多返回值和省略返回的特性**。Go语言的函数定义语法如下所示：
```go
func function_name( [parameter list] ) [return_types] {
  //函数体
}
```
函数定义解析：
- func：函数由 *func* 关键字进行声明。
- function_name：函数名，和参数列表一起构成了函数签名。
- parameter list：参数列表，它指定的是参数类型、顺序、及参数个数。参数是可选的，也就是说函数也可以不包含参数；处于最末尾的参数可以是变长参数，使用 `name ... *type*` 进行声明，使用 `call_func(a, b, c)` （当a、b、c是基本类型）或 `call_func(list...)`（list是数组或切片类型）进行调用。
- return_types：返回类型，**函数可以返回一列值**，return_types 是该列值的数据类型。Golang 也支持对返回值进行命名。对于可能出错的函数来说，Go 开发的最佳实践是将 `error` 型变量作为最后一个返回值。
- 函数体：函数定义的代码集合。

#### 匿名函数
除了标准形式的函数定义之外，**Golang 中的函数定义也可以是以下的匿名函数形式**，可以实现函数的简单嵌套：
```go
var := func( [parameter list] ) [return_types] {
  //函数体
}
```
对于嵌套的匿名函数，在实践中我发现它的特性如下：
- 可以进行简单的函数嵌套，但不可用于自我递归；
- 可以捕获外层函数的局部变量，实现了语言的闭包；
- 可以在声明后加上参数进行当场调用；
综上，使用匿名函数进行重复、简单、相互独立的数据处理是非常合适的，配合 goroutine 和 channel 也可以很容易地处理IO密集型操作的并发执行（例如本项目中的大量数据库通信）和CPU密集型操作的并行执行（如本项目中的动态规划选路），详见 Golang 并发部分。

#### Golang 的参数传递
每种语言的参数传递都有其特征，但总体可以分为按值传递、按引用传递两种。**传值的意思是：函数传递的总是原来这个东西的一个副本，一副拷贝，参数传递前后值相同而地址不同；**而传递引用则是将原变量的地址传入。Golang 默认情况下只支持值传递（特殊情况为slice、map、channel类型），那么如果需要对原先的值进行改变，可以使用以下几种方法：
- 使用指针型参数，通过改变原变量的取值对其进行修改；
- 函数返回同类型的变量，通过对原变量赋值实现替换；
- 对于结构类型，可以定义接受结构指针的方法，从而对结构内部变量进行修改；这里使用的是 Golang 中 `structPtr.member = (*structPtr).member` 的特性。

在Golang中，形式参数会作为函数的局部变量来使用，可以使用 `=` 进行赋值。**而 Golang 的特点之二是返回值可以具有变量名，且可以直接使用 `return` 返回当前传出变量的值**。这一特点对于以下典型场景尤其方便：函数返回多个值，其中之一为 `err error`，且当 `err ！= nil`，其他值无意义。 我们可以简单地为 `err` 变量赋值并直接 `return`。

### 包管理
Golang 的包是以文件夹为单位的，按 `package` 方式组织，再通过 `import` 引入使用。`package` 需要出现在一个文件除去注释外的第一行，`import` 的语法如下所示：
```go
import (
  "fmt" // 标准库导入

  "github.com/someone/somgthing" // 第三方库导入、
  alter "github.com/someone/somgthing2" // 使用别名导入
  . "github.com/someone/somgthing3" // 点导入，调用函数可以省略包名，test常用
)
```
除却标准包可以通过短路径导入外，包的导入路径是基于工作目录，`vendor` 或者 `$GOPATH` 的，在其他方法找不到包的情况下，编译器会在目录 `$GOPATH/src` 子目录下按照给定路径查找包。

在使用 Golang 包声明和引入的过程中，我认为以下几点需要注意：
- 由于不同的文件夹下是不同的包，导致访问同一项目的不同文件夹需要访问 exported 结构体和函数，需要在声明时注意包依赖关系，或者简单将所有的源文件置于一个目录下；
- 在使用 godep 工具建立 `vendor` 文件夹后，可能会出现 `vendor` 中的结构体和 `$GOPATH` 同名同包之间完全相同的结构体不兼容的情况，这时需要删除其一；
- 一个目录中的所有源文件必须处于同一个包中；
- 一个文件夹内有且只能有一个源文件使用 `package main` 声明。

## 数据结构
### 结构体
结构体是由一系列具有相同类型或不同类型的数据构成的数据集合，其定义需要使用 `type` 和 `struct` 语句。`struct` 语句定义一个新的数据类型，结构体有中有一个或多个成员。`type` 语句设定了结构体的名称。*结构体成员声明可以在最后附上字段tag，方便使用反射获得结构体成员说明，在结构体序列化如JSON、BSON与结构体互转中非常常见*。结构体声明的格式如下：
```go
type struct_variable_type struct {
   member definition `tag1:"...", tag2:"..."`;
   member definition `tag1:"...", tag2:"..."`;
   ...
   member definition `"tag1:...", tag2:"..."`;
}
```
一旦定义了结构体类型，它就能用于变量的声明，语法格式如下：
```go
variable_name := structure_variable_type {value1, value2...valuen} // unkeyed
variable_name := structure_variable_type { key1: value1, key2: value2..., keyn: valuen} // keyed
```
我们也可以将以上两部合并，使用匿名结构体来定义只使用一次的结构，如单元测试中的各个输入输出值，如下所示：
```go
// net/sample_test.go:9
type args struct {
  wholeList []Asset
  rate      float64
}
tests := []struct {
  name                 string
  args                 args
  wantSampledIndexList []int
}{{
  name: "Minimal",
  args: args{
    wholeList: []Asset{
      Asset{"A", "base", 0.4, 0.2, 1},
    },
    rate: 1.0,
  },
  wantSampledIndexList: []int{0},
}}
```
如果要访问结构体成员，需要使用点号 `.` 操作符。**在 Golang 中，编译器会自动将 `structPtr.member` 翻译为 `(*structPtr).member`，不存在 `->` 操作符**。

同时需要注意的是，**在参数传递中，结构体是作为一个整体来拷贝的**。所以使用结构体而非指针作为形参的函数以及将结构体而非指针作为接受对象的方法将不能够改变实参的成员值。

### 切片
Golang 切片是对数组的抽象，**切片相比于数组可以动态增长，灵活地进行追加和截取**，切片的定义语法可以为以下的任意一种：
```go
var slice0 []type // 不需要像数组一样指明长度
slice1 := make([]type, len, [capicity]) // 使用make函数
slice2 := []type{} //空列表初始化
```
切片具有数组不具有的灵活性，而且可以作为引用进行参数传递，这是因为切片中的值是对底层数组结构的引用，这也导致了引用同一个底层数组的不同切片操作会造成数据污染。*并且同使用 C++ `std::vector` 一样，在已知大致数据规模的情况下，可以在初始化切片时制定较大的长度，避免重复申请资源造成内存不断拷贝的开销*。

### Map
Map 是一种无序的键值对的集合，其最重要的一点是通过 key 来快速检索数据。Golang 将以哈希表作为底层的 Map 集成为一种基本结构：它首先是个动态、mutable 的，也就是说，可以随时对其进行修改；其次，它不是线程安全的。所以它等价于 Java 里的 HashMap。Map 的定义如下所示：
```golang
var map_variable map[key_data_type]value_data_type //声明变量，默认 map 是 nil
map_variable := make(map[key_data_type]value_data_type) //使用 make 函数
```
Map 可以简单地使用和 Python 类似的语法 `map[key] = v` 进行插入；它的取数据方法为：`v, ok := map[key]`；同时它具有 `delete()` 函数用于删除集合的元素：`delete(map, key)`。Map同样可以做为引用型参数进行传递。

#### 集合
由于 Map 中的键值具有确定性、无序性、互异性，所以可以使用一个 Map 的所有键构成一个集合，值类型任取。例如，可以使用 `string` 作为键类型，存储开销最低的 `bool` 作为值类型，构成一个字符串集合`str_set := make(map[string]bool)`。
在本项目中，考虑到对超过15个点的空间求解TSP问题会造成很大的时间和空间开销，我希望尽可能多地重复利用以往的计算结果。而恰好为了符合现实情况，我将一整个空间划分为层次化的小空间，所以我可以对每个子空间的选路结果进行缓存。考虑到每次选路是整体样本点的一部分，我需要用一个集合定义进行选路的样本点并进行序列化（因为 Map 的键要求必须是可以比较的），以此能够利用 Redis 数据库作为选路缓存。我实现这一集合并利用它获取 cache 的方法如下：
```go
// net/route.go:37
setToString := func(m map[dataio.Checkpoint]bool) string {
  // serialization : map -> string
  s := "{"
  for k, _ := range m {
    s += fmt.Sprintf("%v, ", k)
  }
  return s[:len(s)-2] + "}"
}

// net/route.go:61
cpList := pack(rootPtr.Assets, validSpaceList) // a struct list
keySet := make(map[dataio.Checkpoint]bool) // to construct a set Data Strcture as key of Redis
for _, item := range cpList {
  keySet[item] = true
}

// net/route.go:70
k := setToString(keySet)
data, err := redis.Bytes(redisConn.Do("GET", "route-"+k))
```

### 接口
Golang 提供了接口类型，它把所有的具有共性的方法定义在一起，任何其他类型只要实现了这些方法就是实现了这个接口。它的定义和实现方法如下：
```go
/* 定义接口 */
type interface_name interface {
   method_name1 [return_type]
   method_name2 [return_type]
   method_name3 [return_type]
   ...
   method_namen [return_type]
}

/* 定义结构体 */
type struct_name struct {
   /* variables */
}

/* 实现接口方法 */
func (struct_name_variable struct_name) method_name1() [return_type] {
   /* 方法实现 */
}
...
func (struct_name_variable struct_name) method_namen() [return_type] {
   /* 方法实现*/
}
```
可见 Golang 的接口类型是基于函数表实现的，也无法实现继承，这与大部分面向对象的语言大不相同。

### 通道
通道（channel）是用来传递数据的一个数据结构，可用于两个 goroutine 之间通过传递一个指定类型的值来同步运行和通讯。操作符 `<-` 用于指定通道的方向、发送或接收。如发送：`chan <- data` 和接收：`data := <-chan`。声明一个通道的语法如下：

```go
ch := make(chan int, [buffer length])
```

需要注意的是，在未指明通道缓冲区大小的默认情况下，通道是不带缓冲区的。发送端发送数据，同时必须又接收端相应的接收数据，否则会产生阻塞。带缓冲区的通道允许发送端的数据发送和接收端的数据获取处于异步状态，发送端发送的数据会被拷贝到缓冲区里面，可以等待接收端去获取数据，而不是立刻需要接收端去获取数据。

### 范围
Golang 中 `range` 关键字用于 `for` 循环中迭代数组(array)、切片(slice)、通道(channel)或集合(map)的元素。在数组和切片中它返回元素的索引和索引对应的值，在集合中返回 key-value 对的 key 值，在通道中获取通道里的值和通道的开启情况。在实际开发中，我们更可以结合匿名数组使用 `i, v := range []type{...}` 简洁地对一个短列表中的值进行迭代，实现与 Python 语法中 `in` 关键字类似的效果。

## 并发控制
Golang 原生对并发有着很好的支持，我们可以使用  `go` 关键字和一个函数调用开启一个 goroutine，如 `go func(){...}()` goroutine 是轻量级线程，它的调度是由 Golang 运行时进行管理的。同时 Golang 的 `sync` 包下也为我们提供了许多实用的 goroutine 拓展，如 `errorgroup`、`WaitGroup`等。

### 使用通信进行同步
由于 goroutine 之间是异步进行的，造成了可能的数据竞争和死锁，所以在进行并发编程时，需要对 goroutine 之间的同步机制进行设计。Golang 中，处理并发数据访问的推荐方式是使用管道从一个 goroutine 中往下一个 goroutine 传递实际的数据。有格言说得好：“不要通过共享内存来通讯，而是通过通讯来共享内存”，应用如下所示：
```go
ch := make(chan int)
go func() {
    n := 0 // 仅为一个goroutine可见的局部变量.
    ch <- n // 数据从一个goroutine离开...
}()
n := <-ch   // ...然后安全到达另一个goroutine.
```

### 使用锁进行同步
有时，通过显式加锁，而不是使用管道，来同步数据访问，可能更加便捷。Go语言标准库为这一目的提供了一个互斥锁 - `sync.Mutex`。而要想这类加锁起效的话，关键之处在于：所有对共享数据的访问，不管读写，仅当 goroutine 持有锁才能操作，因为一个goroutine出错就足以破坏掉一个程序，引入数据竞争。因此，应该设计一个自定义数据结构，具备明确的API，确保所有的同步都在数据结构内部完成。如标准库 `sync/WaitGroup` 中提供的三种方法：`Add()` 对锁自增，`Done()` 对锁自减，`Wait()` 等待锁全部释放。

### 避免数据竞争
在循环进行大量 goroutine 操作时，我们需要注意它们的数据操作是否是线程安全的。当 goroutine 对切片、Map 等引用结构进行操作，或是在闭包中捕获了外部变量时，都可能会产生脏数据。在开发过程中我发现可以有以下的解决之道：
- 对闭包函数进行写操作的变量：
  - 使用一个新变量如 `inner := outer` 进行捕获；
  - 作为形参在函数调用中传入；
  - 使用通道作为线程间的数据通路；
- 在 debug 时使用 `go run -race` 进行数据竞争自动检测。

### 并行计算
在现代多核 CPU 上，除了避免 IO 密集型任务先后执行造成的大量等待外，使用并发的另一个优势是可以将 CPU 密集型任务拆分分散到多个核心上执行。在 Go 1.5 版本后，`runtime.GOMAXPROCS` 默认值即是 CPU 的总核心数，因此我们对计算密集型任务进行以下的合理划分：
- 每个工作单元应该花费大约100微秒到1毫秒的时间用于计算。如果单元粒度太小，切分问题以及调度子问题的管理开销可能就会太大。如果单元粒度太大，整个计算也许不得不等待一个慢的工作项结束。这种缓慢可能因为多种原因而产生，比如：调度、其他进程的中断或者糟糕的内存布局。（注意：工作单元的数目是不依赖于CPU的数目的）
- 尽可能减小共享的数据量。并发写操作的代价非常大，特别是如果goroutine运行在不同的CPU上。读操作之间的数据共享则通常不会是个问题。
- 数据访问尽量利用良好的局部性。如果数据能保持在缓存中，数据加载和存储将会快得多得多，这对于写操作也格外地重要。

在本项目中，我利用了层次化空间的概念，将选路算法部分并行化了，思路如下：
- 需要进行选路的所有资产点和子空间可以使用一个以母空间为根节点的树结构组织起来；
- 由于需要进行采样，所以需要首先自上而下地进行节点的搜索，并对其中的资产点进行抽样；但是一个空间在进行过采样后，可能已经不包含任意一个资产点，所以不需要进行选路；
- 因此，需要在采样后以后根遍历的方式，自下而上地进行选路：
  - 首先，对每个子空间节点进行判空。若不空，则进行递归选路；否则返回空；
  - 根节点判断是否每个子节点为空。若不空，则解本空间中所有资产点和子空间点作为集合的 TSP 问题；否则返回空
因此我们可以看到，在这个问题上，判空存在空间之间的依赖关系；而在不空的条件下如何选路则是相互独立的问题。所以我们可以把求解 TSP 问题使用 goroutine 并行化，并使用 `WaitGroup` 等待所有空间节点的选路完成后进行合并等操作。此流程在 `net/route.go` 中实现，代码大致如下所示：
```go
// net/route.go:35
// post-order traversal to sample and dispatch routing task
func (r RestContext) recursiveSampleTSP(rootPtr *spaceNaviNode) bool { 
  // T/F : the sub-tree contains Assets after sampling -> need to routine or not
  validSpaceList := []Space{}
  // filter the subtrees
  for _, subNode := range rootPtr.subspaces {
    if r.recursiveSampleTSP(subNode) { // have checkpoints
      validSpaceList = append(validSpaceList, subNode.root)
    }
  }

  if len(rootPtr.Assets) == 0 && len(validSpaceList) == 0 { // empty Asset list, empty sub trees
    return false
  }

  // could do computing in parallel
  wgTSP.Add(1) // wgTSP: WaitGroup for solving TSP, global var

  go func() {
    // the routing point list
    cpList := pack(rootPtr.Assets, validSpaceList) // must not be empty

    if **could find from Redis cache** {
      wgTSP.Done()
      return true
    }
    
    **call TSP()**
    go **cache the result**
    wgTSP.Done()
  }()

  return true
}

// net/route.go:189
// in calcRoute():
r.recursiveSampleTSP(masterRootPtr) // TSP bottom to up
wgTSP.Wait()                        // until all computations compelete
...
```

## 错误处理
错误指的是可能出现问题的地方出现了问题，比如打开一个文件时失败，这种情况在人们的意料之中 ；而异常指的是不应该出现问题的地方出现了问题，比如引用了空指针，这种情况在人们的意料之外。可见，**错误是业务过程的一部分，而异常不是**。同时错误和异常是可以互相转换的：如程序逻辑上重试产生错误的操作次数过多，可以升级为异常；而 `panic` 触发的异常被 `recover` 恢复后，可以将返回值中 `error` 类型的变量进行赋值，以便上层函数继续走错误处理流程。

Golang 秉承着“少即是多”的哲学，并不提倡将一切错误都作为异常抛出，而是引入 `error` 接口类型作为错误处理的标准模式；引入两个内置函数 `panic` 和 `recover` 来触发和终止异常处理流程，同时引入关键字 `defer` 来延迟执行 `defer` 后面的函数，作为异常处理的标准模式。

在开发过程中，我需要对网络 I/O 超时、本地文件操作，和用户输入逻辑冲突等各种各样的错误和异常进行处理，经过学习和自己的实践，我发现了以下较好的错误处理实践：
- 失败的原因只有一个时，不使用error，而使用 `bool` 型返回值，这样可以简化程序逻辑。
- 错误值统一定义，而不是跟着感觉走。使用 `errors.New()` 函数可以很快地创建一个错误，但这会造成命名不规范、错误一层层上传造成log格式混乱等。于是，我们可以参考C/C++的错误码定义文件，在Golang的每个包中增加一个错误对象定义文件。
- 错误逐层传递时，层层都加日志，显示当前的错误位置，方便 debug 定位。
- 使用 `defer` 处理产生错误后应当释放掉的资源，如 `defer func() {if err != nil {...}}()`。这是由于 `defer` 后的函数会在函数退出时执行，而闭包函数捕获的 `err` 始终是函数返回时最新的引用，所以可以判断返回时是否出现了错误。并且 `defer` 按照 LIFO 的原则依次调用，故可以处理层级性的资源释放。
- goroutine 中应当避免出现异常，因为 goroutine 中的 `panic` 会直接传至主携程，造成整个程序运行的中止。

实践以上规则，我在 `net/restful.go` 中对配置环境进行初始化的错误处理流程如下所示：
```go
// net/restful.go:38
// InitEnv : check and try to correct the RestContext and connet to DB servers
func (r *RestContext) InitEnv() (err error) {

  // 在函数出口监测错误
	defer func() { // global recover
		if err != nil { // using clousure to detect err's return value
			log.Println("all tries failed")
			debug.PrintStack()
		}
	}()
  
	if _, err = os.Stat(r.CrtPath); err != nil {
		log.Println(err)
		log.Println("crt file not exist, trying BakCtx crt file")
		r.CrtPath = BakCtx.CrtPath
		if _, err := os.Stat(r.CrtPath); err != nil {
			log.Println(err)
			log.Println("crt file not exist, trying RCTest crt file")
			r.CrtPath = RCTest.CrtPath
			if _, err := os.Stat(r.CrtPath); err != nil {
				log.Println(err)
				log.Println("crt file not exist again")
				return err
			}
		}
	}

  ...

	return nil
}
```