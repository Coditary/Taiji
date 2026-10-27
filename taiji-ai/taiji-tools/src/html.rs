pub fn html_to_text(input: &str) -> String {
    html2text::from_read(input.as_bytes(), 120)
        .map(|s| s.trim().to_string())
        .unwrap_or_else(|_| input.trim().to_string())
}
