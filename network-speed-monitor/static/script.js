// Function to fetch stats from the backend and update the UI
function fetchStats() {
    fetch('/api/stats')
        .then(response => response.json())
        .then(data => {
            const statusDiv = document.getElementById('status');
            statusDiv.innerHTML = `
                <p><strong>Timestamp:</strong> ${data.timestamp}</p>
                <p><strong>Connectivity:</strong> ${data.connectivity ? "Online" : "Offline"}</p>
                <p><strong>Upload Speed:</strong> ${data.upload_speed.toFixed(2)} Mbps</p>
                <p><strong>Download Speed:</strong> ${data.download_speed.toFixed(2)} Mbps</p>
            `;

            // Append downtime data to the table if necessary
            if (!data.connectivity || data.upload_speed < 2 || data.download_speed < 2) {
                const table = document.getElementById('downtime-logs');
                const row = table.insertRow();
                row.innerHTML = `
                    <td>${data.timestamp}</td>
                    <td>${data.connectivity ? "Bandwidth Low" : "Offline"}</td>
                    <td>-</td>
                `;
            }
        })
        .catch(error => {
            console.error('Error fetching stats:', error);
        });
}

// Fetch stats every 5 seconds
setInterval(fetchStats, 5000);

// Fetch stats immediately when the page loads
fetchStats();
