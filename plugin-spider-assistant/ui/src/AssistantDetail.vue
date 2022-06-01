<template>
  <div class="assistant-detail">
    <div class="top-bar">
      <cl-form
          :model="spiderData"
          :grid="3"
      >
        <cl-form-item :span="1" :label="t('assistant.detail.framework')">
          <cl-tag
              :label="spiderDataFrameworkLabel"
              :type="spiderDataFrameworkType"
              size="normal"
          />
        </cl-form-item>
      </cl-form>
    </div>
    <div class="spider-content">
      <template v-if="spiderData.framework === 'scrapy'">
        <Scrapy/>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import {computed, defineComponent, onMounted, ref} from 'vue';
import {useRequest} from 'crawlab-ui';
import {useRoute} from 'vue-router';
import Scrapy from './scrapy/Scrapy.vue';

const {
  get,
} = useRequest();

const endpoint = '/plugin-proxy/spider-assistant';

const pluginName = 'spider-assistant';
const t = (path) => window['_tp'](pluginName, path);

export default defineComponent({
  name: 'AssistantDetail',
  components: {Scrapy},
  setup(props, {emit}) {
    const route = useRoute();

    const spiderData = ref({
      framework: '',
    });

    const getSpiderData = async () => {
      const id = route.params.id;
      if (!id) return;
      const res = await get(`${endpoint}/spiders/${id}`);
      const {data} = res;
      spiderData.value = data;
    };

    onMounted(getSpiderData);

    const spiderDataFrameworkLabel = computed(() => {
      switch (spiderData.value.framework) {
        case 'scrapy':
          return t('assistant.detail.scrapy');
        default:
          return t('assistant.detail.noFramework');
      }
    });

    const spiderDataFrameworkType = computed(() => {
      switch (spiderData.value.framework) {
        case 'scrapy':
          return 'primary';
        default:
          return 'info';
      }
    });

    return {
      spiderData,
      spiderDataFrameworkLabel,
      spiderDataFrameworkType,
      t,
    };
  },
});
</script>

<style scoped>
.assistant-detail .top-bar {
  padding: 10px 0;
  border-bottom: 1px solid #e6e6e6;
}

.assistant-detail .top-bar >>> .el-form-item {
  margin-bottom: 0;
}
</style>
