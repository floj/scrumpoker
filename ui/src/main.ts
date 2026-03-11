import { createApp } from 'vue';

import 'bootstrap/scss/bootstrap.scss';
import "bootstrap-icons/font/bootstrap-icons.scss";
import { Dropdown } from 'bootstrap';

import App from './App.vue';
import router from './router';

import 'notyf/notyf.min.css';


const app = createApp(App);

app.use(router);
app.mount('#app');
