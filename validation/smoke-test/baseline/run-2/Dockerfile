FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create uploads directory
RUN mkdir -p /app/uploads

# Expose port 5000 for Flask
EXPOSE 5000

# Default command (can be overridden in docker-compose)
CMD ["flask", "run", "--host=0.0.0.0"]
