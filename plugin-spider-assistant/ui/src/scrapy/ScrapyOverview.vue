<template>
  <cl-form class="scrapy-overview">
    <cl-form-item :span="2" :label="t('scrapy.navItems.settings')">
      <cl-tag v-for="(d, $index) in settings" :key="$index" :label="d"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('scrapy.overview.deploy')">
      <cl-tag v-for="(d, $index) in deploy" :key="$index" :label="d"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('scrapy.navItems.spiders')">
      <cl-tag :label="getCount('spiders')" clickable @click="onGoto('spiders')"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('scrapy.navItems.items')">
      <cl-tag :label="getCount('items')" clickable @click="onGoto('items')"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('scrapy.navItems.middlewares')">
      <cl-tag :label="getCount('middlewares')" clickable @click="onGoto('middlewares')"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('scrapy.navItems.settings')">
      <cl-tag :label="getCount('settings')" clickable @click="onGoto('settings')"/>
    </cl-form-item>
  </cl-form>
</template>

<script lang="ts">
import {computed, defineComponent} from 'vue';

const pluginName = 'spider-assistant';
const t = (path) => window['_tp'](pluginName, path);

export default defineComponent({
  name: 'ScrapyOverview',
  props: {
    form: {
      type: Object,
      default: () => {
        return {};
      },
    },
  },
  emit: [
    'goto',
  ],
  setup(props, {emit}) {
    const settings = computed(() => {
      const {cfg} = props.form;
      const {settings} = cfg || {};
      if (!settings) return [];
      const arr = [];
      for (const key in settings) {
        const value = settings[key];
        arr.push(`${key}: ${value}`);
      }
      return arr;
    });

    const deploy = computed(() => {
      const {cfg} = props.form;
      const {deploy} = cfg || {};
      if (!deploy) return [];
      const arr = [];
      for (const key in deploy) {
        const value = deploy[key];
        arr.push(`${key}: ${value}`);
      }
      return arr;
    });

    const getCount = (key) => {
      const d = props.form[key];
      if (!d) {
        return '0';
      }
      return (d.length || 0).toString();
    };

    const onGoto = (key) => {
      emit('goto', key);
    };

    return {
      settings,
      deploy,
      getCount,
      onGoto,
      t,
    };
  },
});
</script>

<style scoped>
.scrapy-overview {
  margin: 10px 0;
}
</style>
