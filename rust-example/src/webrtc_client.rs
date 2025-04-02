use std::process::Command;

pub fn join_session(session_id: &str) -> bool {
    let output = Command::new("ffi/ffi.exe")      // Path to ffi.exe (inside the ffi folder).
        .arg("join_session")                      // Action to perform: join_session.
        .arg(session_id)                          // Passing the session ID.
        .output();                                // Running the command.

    match output {
        Ok(output) => output.status.success(),    // Return true if successful.
        Err(_) => false,                          // Return false if error occurs.
    }
}

pub fn send(target_id: &str, message: &str) -> bool {
    let output = Command::new("ffi/ffi.exe")      // Path to ffi.exe.
        .arg("send")                              // Action: send.
        .arg(target_id)                            // Passing target ID.
        .arg(message)                              // Passing the message.
        .output();

    match output {
        Ok(output) => output.status.success(),    // Return true if successful.
        Err(_) => false,                          // Return false if error occurs.
    }
}

pub fn close() -> bool {
    let output = Command::new("ffi/ffi.exe")      // Path to ffi.exe.
        .arg("close")                             // Action: close.
        .output();

    match output {
        Ok(output) => output.status.success(),    // Return true if successful.
        Err(_) => false,                          // Return false if error occurs.
    }
}
