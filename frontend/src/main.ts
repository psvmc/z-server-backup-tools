import { createApp } from "vue";
import Antd from "ant-design-vue";
import "ant-design-vue/dist/reset.css";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import App from "./App.vue";
import "./style.css";

dayjs.locale("zh-cn");
createApp(App).use(Antd).mount("#app");
