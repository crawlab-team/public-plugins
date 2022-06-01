<template>
  <cl-nav-tabs
      :active-key="activeKey"
      :items="navItems"
      @select="onActiveKeyChange"
  />
  <div class="scrapy-content">
    <template v-if="activeKey === 'overview'">
      <ScrapyOverview :form="form" @goto="onActiveKeyChange"/>
    </template>
    <template v-if="activeKey === 'spiders'">
      <ScrapySpiders :form="form"/>
    </template>
    <template v-if="activeKey === 'items'">
      <ScrapyItems :form="form"/>
    </template>
    <template v-if="activeKey === 'middlewares'">
      <ScrapyMiddlewares :form="form"/>
    </template>
    <template v-if="activeKey === 'settings'">
      <ScrapySettings :form="form"/>
    </template>
  </div>
</template>

<script lang="ts">
import {computed, defineComponent, onMounted, ref} from 'vue';
import ScrapyOverview from './ScrapyOverview.vue';
import ScrapySpiders from './ScrapySpiders.vue';
import ScrapyItems from './ScrapyItems.vue';
import ScrapySettings from './ScrapySettings.vue';
import ScrapyMiddlewares from './ScrapyMiddlewares.vue';
import {useRoute} from 'vue-router';
import {useRequest} from 'crawlab-ui';

const pluginName = 'spider-assistant';
const t = (path) => window['_tp'](pluginName, path);

const {
  get
} = useRequest();

const endpoint = '/plugin-proxy/spider-assistant';

export default defineComponent({
  name: 'Scrapy',
  components: {ScrapyMiddlewares, ScrapySettings, ScrapyItems, ScrapySpiders, ScrapyOverview},
  props: {},
  setup(props, {emit}) {
    const route = useRoute();

    const form = ref({});

    const activeKey = ref('overview');

    const navItems = computed(() => [
      {id: 'overview', title: t('scrapy.navItems.overview')},
      {id: 'spiders', title: t('scrapy.navItems.spiders')},
      {id: 'items', title: t('scrapy.navItems.items')},
      {id: 'middlewares', title: t('scrapy.navItems.middlewares')},
      {id: 'settings', title: t('scrapy.navItems.settings')},
    ]);

    const getForm = async () => {
      const id = route.params.id;
      if (!id) return;
      const res = await get(`${endpoint}/scrapy/${id}`);
      const {data} = res;
      form.value = data;
    };

    onMounted(getForm);

    const onActiveKeyChange = (key) => {
      activeKey.value = key;
    };

    return {
      activeKey,
      navItems,
      form,
      getForm,
      onActiveKeyChange,
    };
  },
});
</script>

<style scoped>
.scrapy-content {
}
</style>
