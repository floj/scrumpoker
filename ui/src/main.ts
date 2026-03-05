import { createApp } from 'vue';

import App from './App.vue';
import router from './router';

import 'notyf/notyf.min.css';

const app = createApp(App);

app.config.errorHandler = (err, _instance, info) => {
  console.error(`Unhandled error in ${info}:`, err);
};

app.use(router);
app.mount('#app');
