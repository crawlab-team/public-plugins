<template>
  <cl-transfer
      :titles="titles"
      :data="triggerList"
      :value="enabled"
      @change="onChange"
  />
</template>

<script lang="ts">
import {computed, defineComponent, ref} from 'vue';

const pluginName = 'notifications';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

export default defineComponent({
  name: 'NotificationDetailTabTriggers',
  props: {
    form: {
      type: Object,
      default: () => {
        return {};
      },
    },
    triggerList: {
      type: Array,
      default: () => {
        return [];
      },
    }
  },
  emits: [
    'change',
  ],
  setup(props, {emit}) {
    const titles = computed(() => [
      _t('components.transfer.titles.available'),
      _t('components.transfer.titles.enabled'),
    ]);

    const enabled = computed(() => {
      const {triggers} = props.form;
      return triggers || [];
    });

    const onChange = (value) => {
      emit('change', value);
    };

    return {
      titles,
      enabled,
      onChange,
    };
  },
});
</script>

<style scoped>

</style>
