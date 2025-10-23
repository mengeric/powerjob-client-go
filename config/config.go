package config

// Config 组件运行所需的完整配置（可选）。
// 功能：承载 HTTP 监听（由宿主使用）、数据库与与 PowerJob-Server 通讯相关配置。
// 注意：组件本身不创建 HTTP 服务；Host/Port 供宿主参考。
type Config struct {
    Host string // 服务监听地址，例如 0.0.0.0
    Port int    // 服务监听端口，例如 27777

    Mysql struct {
        DataSource string // 形如 user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=true&loc=Local
    }

    BootstrapServer  string
    HeartbeatSeconds int
    ReportSeconds    int
    DiscoverySeconds int
    AppName          string
    ClientVersion    string
}

