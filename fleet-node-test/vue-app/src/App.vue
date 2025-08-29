<template>
  <div id="app">
    <h1>Vue.js Test Application</h1>
    <p>Running with Fleet Node.js runtime</p>
    
    <div class="info">
      <h2>Environment Info:</h2>
      <ul>
        <li>API URL: {{ apiUrl }}</li>
        <li>Port: 8080 (Vue default)</li>
      </ul>
    </div>

    <div class="data">
      <h2>Sample Data:</h2>
      <button @click="fetchData">Fetch Users</button>
      <pre v-if="users">{{ JSON.stringify(users, null, 2) }}</pre>
    </div>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      apiUrl: process.env.VUE_APP_API_URL || 'http://localhost:3000',
      users: null
    }
  },
  methods: {
    async fetchData() {
      try {
        const response = await fetch(`${this.apiUrl}/api/users`);
        this.users = await response.json();
      } catch (error) {
        console.error('Failed to fetch users:', error);
        this.users = { error: 'Failed to fetch data' };
      }
    }
  }
}
</script>

<style>
#app {
  font-family: system-ui, -apple-system, sans-serif;
  padding: 2rem;
  max-width: 800px;
  margin: 0 auto;
}

.info, .data {
  margin-top: 2rem;
  padding: 1rem;
  background: #f5f5f5;
  border-radius: 8px;
}

button {
  padding: 0.5rem 1rem;
  background: #42b883;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

button:hover {
  background: #3aa876;
}

pre {
  background: white;
  padding: 1rem;
  border-radius: 4px;
  margin-top: 1rem;
}
</style>