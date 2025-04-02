use std::process::Command;
use std::sync::{Arc, Barrier};
use std::thread;

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

fn user_1(session_id: &str, barrier: Arc<Barrier>) {
    barrier.wait();  // Wait for synchronization
    println!("User 1: Joining session...");
    if join_session(session_id) {
        println!("User 1: Successfully joined session.");
    } else {
        println!("User 1: Failed to join session.");
        return;
    }

    // Simulate User 1 sending a message to User 2 after joining
    println!("User 1: Sending message...");
    if send("user_2_id", "Hello from User 1") {
        println!("User 1: Message sent.");
    } else {
        println!("User 1: Failed to send message.");
    }

    // Close User 1's session
    println!("User 1: Closing session...");
    if close() {
        println!("User 1: Session closed.");
    } else {
        println!("User 1: Failed to close session.");
    }
}

fn user_2(session_id: &str, barrier: Arc<Barrier>) {
    barrier.wait();  // Wait for synchronization
    println!("User 2: Joining session...");
    if join_session(session_id) {
        println!("User 2: Successfully joined session.");
    } else {
        println!("User 2: Failed to join session.");
        return;
    }

    // Simulate User 2 receiving a message from User 1
    println!("User 2: Sending response message...");
    if send("user_1_id", "Hello from User 2") {
        println!("User 2: Message sent.");
    } else {
        println!("User 2: Failed to send message.");
    }

    // Close User 2's session
    println!("User 2: Closing session...");
    if close() {
        println!("User 2: Session closed.");
    } else {
        println!("User 2: Failed to close session.");
    }
}

fn main() {
    let session_id = "test_session_123";
    let barrier = Arc::new(Barrier::new(2));  // Barrier for 2 threads

    // Start both user threads
    let user_1_thread = thread::spawn({
        let barrier = Arc::clone(&barrier);  // Clone the Arc to move into the thread
        move || {
            user_1(session_id, barrier);
        }
    });
    
    let user_2_thread = thread::spawn({
        let barrier = Arc::clone(&barrier);  // Clone the Arc to move into the thread
        move || {
            user_2(session_id, barrier);
        }
    });

    // Wait for both threads to complete
    user_1_thread.join().unwrap();
    user_2_thread.join().unwrap();
}
