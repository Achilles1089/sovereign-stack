import SwiftUI

struct ContentView: View {
    @EnvironmentObject var server: ServerManager
    
    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            
            VStack(spacing: 20) {
                // Header
                Text("SOVEREIGN-OS")
                    .font(.system(.title, design: .monospaced))
                    .foregroundColor(.green)
                
                Text("iPhone Inference Node")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundColor(.green.opacity(0.6))
                
                Divider().background(Color.green.opacity(0.3))
                
                // Server Status
                VStack(alignment: .leading, spacing: 8) {
                    statusRow("STATUS", value: server.isRunning ? "ONLINE" : "OFFLINE",
                              color: server.isRunning ? .green : .red)
                    statusRow("ADDRESS", value: server.address)
                    statusRow("PORT", value: "\(server.port)")
                    statusRow("MODEL", value: server.activeModel)
                    statusRow("SPEED", value: server.lastTokPerSec > 0 ? 
                              String(format: "%.1f tok/s", server.lastTokPerSec) : "—")
                }
                .padding()
                .background(Color.green.opacity(0.05))
                .cornerRadius(4)
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke(Color.green.opacity(0.2), lineWidth: 1)
                )
                
                // Device Info
                VStack(alignment: .leading, spacing: 8) {
                    statusRow("DEVICE", value: deviceName())
                    statusRow("RAM", value: ramInfo())
                    statusRow("BATTERY", value: batteryInfo())
                }
                .padding()
                .background(Color.green.opacity(0.05))
                .cornerRadius(4)
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke(Color.green.opacity(0.2), lineWidth: 1)
                )
                
                // Requests log
                VStack(alignment: .leading, spacing: 4) {
                    Text("REQUESTS")
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundColor(.green.opacity(0.5))
                    
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 2) {
                            ForEach(server.logs.suffix(20), id: \.self) { log in
                                Text(log)
                                    .font(.system(size: 10, design: .monospaced))
                                    .foregroundColor(.green.opacity(0.7))
                            }
                        }
                    }
                    .frame(maxHeight: 150)
                }
                .padding()
                .background(Color.green.opacity(0.05))
                .cornerRadius(4)
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .stroke(Color.green.opacity(0.2), lineWidth: 1)
                )
                
                Spacer()
                
                // cURL hint
                Text("curl http://\(server.address):\(server.port)/v1/models")
                    .font(.system(size: 10, design: .monospaced))
                    .foregroundColor(.green.opacity(0.3))
                    .textSelection(.enabled)
            }
            .padding()
        }
        .preferredColorScheme(.dark)
    }
    
    func statusRow(_ label: String, value: String, color: Color = .green) -> some View {
        HStack {
            Text(label)
                .font(.system(size: 11, design: .monospaced))
                .foregroundColor(.green.opacity(0.5))
                .frame(width: 80, alignment: .leading)
            Text(value)
                .font(.system(size: 13, design: .monospaced))
                .foregroundColor(color)
        }
    }
    
    func deviceName() -> String {
        var systemInfo = utsname()
        uname(&systemInfo)
        let machine = withUnsafePointer(to: &systemInfo.machine) {
            $0.withMemoryRebound(to: CChar.self, capacity: 1) {
                String(cString: $0)
            }
        }
        return machine
    }
    
    func ramInfo() -> String {
        let total = ProcessInfo.processInfo.physicalMemory
        let totalGB = Double(total) / 1_073_741_824
        return String(format: "%.1fG total", totalGB)
    }
    
    func batteryInfo() -> String {
        UIDevice.current.isBatteryMonitoringEnabled = true
        let level = UIDevice.current.batteryLevel
        if level < 0 { return "Unknown" }
        return "\(Int(level * 100))%"
    }
}
